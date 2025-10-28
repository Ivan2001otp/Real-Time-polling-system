package services

import (
	"RealTimePoll/internal/database"
	"RealTimePoll/internal/models"
	"RealTimePoll/internal/utils"

	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	SessionCacheTTL = 48 * time.Hour
	SESSION_KEY_PREFIX = "session:"
	ORGANIZER_SESSION_KEY = "organizer_sessions:"
	SESSION_BY_JOINCODEKEY = "session_joincode:"
)

func SavePollQuestions(session models.Session) (*string, error) {
	mongoDb := database.GetMongoInstance();
	sessionsCollection := mongoDb.GetCollection(utils.SESSION_COLLECTION);

	ctx := context.Background();

	for i:= range session.Questions {
		session.Questions[i].ID = primitive.NewObjectID();
	}

	// save to mongo
	_, err := sessionsCollection.InsertOne(ctx, session);

	if err != nil {
		return nil, fmt.Errorf("failed to save session to MongoDB: %v", err)
	}

	log.Printf("Session %s saved to MongoDB for organizer %s", session.ID.Hex(), session.OrganizerId.Hex())

	if err := cacheSessionInRedis(&session); err != nil {
		log.Printf("Warning: Failed to cache session in Redis: %v", err)
	}

	idStr := session.ID.Hex();
	return &idStr, nil
}

func cacheSessionInRedis(session *models.Session) error {

	redisDb := database.GetRedisInstance();
	ctx := context.Background();

	sessionKey := SESSION_KEY_PREFIX + session.ID.Hex()
	sessionJSON, err := json.Marshal(session)
    if err != nil {
        return fmt.Errorf("failed to marshal session for Redis: %v", err)
    }

    err = redisDb.GetClient().Set(ctx, sessionKey, sessionJSON, SessionCacheTTL).Err()
    if err != nil {
        return fmt.Errorf("failed to cache session in Redis: %v", err)
    }


	organizerKey := ORGANIZER_SESSION_KEY + session.OrganizerId.Hex();

	sessionInfoJSON, err := json.Marshal(session);
	if err != nil {
        return fmt.Errorf("failed to marshal session info for Redis: %v", err)
    }

	err = redisDb.GetClient().ZAdd(ctx, organizerKey, redis.Z{
        Score:  float64(session.CreatedAt.Unix()),
        Member: sessionInfoJSON,
    }).Err()
    if err != nil {
        return fmt.Errorf("failed to add session to organizer set: %v", err)
    }

	redisDb.GetClient().Expire(ctx, organizerKey, SessionCacheTTL)

    // 3. Store join code to session ID mapping
    joinCodeKey := SESSION_BY_JOINCODEKEY + session.JoinCode
    err = redisDb.GetClient().Set(ctx, joinCodeKey, session.ID.Hex(), SessionCacheTTL).Err()
    if err != nil {
        return fmt.Errorf("failed to cache join code mapping: %v", err)
    }

    log.Printf("Session %s cached in Redis with multiple access patterns", session.ID.Hex())
    return nil
}

func GetSessionsByOrganizer(organizerID primitive.ObjectID) ([]models.Session, error) {
    redisDb := database.GetRedisInstance()
    ctx := context.Background()

    organizerKey := ORGANIZER_SESSION_KEY + organizerID.Hex()
    
    // Get all sessions from sorted set (latest first)
    sessions, err := redisDb.GetClient().ZRevRange(ctx, organizerKey, 0, -1).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to get sessions from Redis: %v", err)
    }

    if len(sessions) == 0 {
        // Fallback to MongoDB
        return getSessionsFromMongoDB(organizerID)
    }

    var sessionList []models.Session
    for _, sessionJSON := range sessions {
        var sessionInfo models.Session
        if err := json.Unmarshal([]byte(sessionJSON), &sessionInfo); err == nil {
            sessionList = append(sessionList, sessionInfo)
        }
    }

    log.Printf("Retrieved %d sessions from Redis for organizer %s", len(sessionList), organizerID.Hex())
    return sessionList, nil
}



func getSessionsFromMongoDB(organizerID primitive.ObjectID) ([]models.Session, error) {
    mongo := database.GetMongoInstance()
    sessionsCollection := mongo.GetCollection("sessions")
    ctx := context.Background()

    cursor, err := sessionsCollection.Find(ctx, map[string]interface{}{
        "organizer_id": organizerID,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to find sessions in MongoDB: %v", err)
    }
    defer cursor.Close(ctx)

    var sessions []models.Session
    if err := cursor.All(ctx, &sessions); err != nil {
        return nil, fmt.Errorf("failed to decode sessions: %v", err)
    }

    // Cache individual sessions in Redis
    for i := range sessions {
        if err := cacheSessionInRedis(&sessions[i]); err != nil {
            log.Printf("Warning: Failed to cache session in Redis: %v", err)
        }
    }

    log.Printf("Retrieved %d sessions from MongoDB for organizer %s", len(sessions), organizerID.Hex())
    return sessions, nil
}


// update status for session poll
func UpdateSessionStatus(sessionID primitive.ObjectID, status string) error {
	mongoDb := database.GetMongoInstance();
	sessionsCollection := mongoDb.GetCollection(utils.SESSION_COLLECTION)
    ctx := context.Background()

	nowTime, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339));
    // Update in MongoDB
    _, err := sessionsCollection.UpdateOne(
        ctx,
        map[string]interface{}{"_id": sessionID},
        map[string]interface{}{
            "$set": map[string]interface{}{
                "status":     status,
                "updated_at": nowTime,
            },
        },
    )
    if err != nil {
        return fmt.Errorf("failed to update session status in MongoDB: %v", err)
    }



	sessionKey := SESSION_KEY_PREFIX + sessionID.Hex();
	redisDb := database.GetRedisInstance();
	redisDb.GetClient().Del(ctx, sessionKey);

	log.Printf("Session %s status updated to %s", sessionID.Hex(), status)
    return nil
}





// GetVoteCountForQuestion returns vote count for a question from Redis cache(for real time display)
func GetVoteCountForQuestion(questionID primitive.ObjectID) (int64, error) {
    redis := database.GetRedisInstance()
    ctx := context.Background()

    questionVotesKey := fmt.Sprintf("question_votes:%s", questionID.Hex())
    count, err := redis.GetClient().SCard(ctx, questionVotesKey).Result()
    if err != nil {
        return 0, fmt.Errorf("failed to get vote count from Redis: %v", err)
    }

    return count, nil
}

// GetVotesForSession returns all vote IDs for a session from Redis(for real time display)
func GetVotesForSession(sessionID primitive.ObjectID) ([]string, error) {
    redis := database.GetRedisInstance()
    ctx := context.Background()

    sessionVotesKey := fmt.Sprintf("session_votes:%s", sessionID.Hex())
    voteIDs, err := redis.GetClient().SMembers(ctx, sessionVotesKey).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to get session votes from Redis: %v", err)
    }

    return voteIDs, nil
}