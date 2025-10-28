package repository;

import (
	"RealTimePoll/internal/models"
	"RealTimePoll/internal/database"
	"RealTimePoll/internal/utils"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"encoding/json"
	"log"
	"context"
	"fmt"
	"time"
)

const (
	SessionCacheTTL = 48 * time.Hour
	SESSION_KEY_PREFIX = "session:"
	ORGANIZER_SESSION_KEY = "organizer_sessions:"
	SESSION_BY_JOINCODEKEY = "session_joincode:"
)

func fetchSessionFromCache(sessionID primitive.ObjectID) (*models.Session, error) {
	redisDb := database.GetRedisInstance();
	ctx := context.Background();

	sessionKey := SESSION_KEY_PREFIX + sessionID.Hex();

	sessionJson, err := redisDb.GetClient().Get(ctx, sessionKey).Result()
	 if err != nil {
        return nil, fmt.Errorf("Session not found in Redis: %v", err)
    }

	var session models.Session

	if err = json.Unmarshal([]byte(sessionJson), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session from Redis: %v", err);
	}

	log.Printf("Session %s retrieved from Redis cache", sessionID.Hex())
    return &session, nil
}


func GetSessionByID(sessionID primitive.ObjectID) (*models.Session, error) {
	var session *models.Session;
	session, err := fetchSessionFromCache(sessionID);
	if err == nil {
		log.Println("Cache Hit- The session is present in redis.");
		return session,nil;
	}

	log.Printf("Cache miss - Session %s not in cache, fetching from MongoDB: %v", sessionID.Hex(), err);
	return fetchSessionFromMongoDB(sessionID);
}

func fetchSessionFromMongoDB(sessionID primitive.ObjectID) (*models.Session, error) {
	mongoDb := database.GetMongoInstance();
    sessionsCollection := mongoDb.GetCollection(utils.SESSION_COLLECTION);

    ctx := context.Background();
    var session models.Session;

    err := sessionsCollection.FindOne(ctx, map[string]interface{}{
        "_id": sessionID,
    }).Decode(&session);

    if err != nil {
        return nil, fmt.Errorf("session not found in MongoDB: %v", err)
    }

    log.Printf("Session %s retrieved from MongoDB", sessionID.Hex())

	if err := cacheSessionInRedis(&session); err != nil {
		log.Printf("Warning: Failed to cache session in Redis: %v", err);
	}

	return &session,nil;
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



// save votes to mongo
func SaveVoteToMongo(vote models.Vote) error {
	mongoDb := database.GetMongoInstance();
	votesCollection := mongoDb.GetCollection(utils.VOTES_COLLECTION);
	ctx := context.Background();

	_, err := votesCollection.InsertOne(ctx, vote);
	if err != nil {
		if isDuplicateKeyError(err) {
            log.Printf("Duplicate vote detected in MongoDB: %v", err)
            return fmt.Errorf("duplicate vote")
        }
        return fmt.Errorf("failed to insert vote: %v", err)
	}

	log.Printf("Vote saved to MongoDB: %s", vote.ID.Hex())

    // Cache the vote in Redis for faster access and real-time processing
    if err := cacheVoteInRedis(vote); err != nil {
        log.Printf("Warning: Failed to cache vote in Redis: %v", err)
        // Don't return error - MongoDB save was successful
    }

    return nil
}


func cacheVoteInRedis(vote models.Vote) error {
    redisDb := database.GetRedisInstance()
    ctx := context.Background();


	voteKey := fmt.Sprintf("vote:%s", vote.ID.Hex())
    voteJSON, err := json.Marshal(vote)
    if err != nil {
        return fmt.Errorf("failed to marshal vote for Redis: %v", err)
    }

	err = redisDb.GetClient().Set(ctx, voteKey, voteJSON, 24*time.Hour).Err()
    if err != nil {
        return fmt.Errorf("failed to cache vote in Redis: %v", err)
    }

	sessionVotesKey := fmt.Sprintf("session_votes:%s", vote.SessionID.Hex())
    err = redisDb.GetClient().SAdd(ctx, sessionVotesKey, vote.ID.Hex()).Err()
    if err != nil {
        return fmt.Errorf("failed to add vote to session set: %v", err)
    }


	redisDb.GetClient().Expire(ctx, sessionVotesKey, 24*time.Hour)

	participantVoteKey := fmt.Sprintf("participant_vote:%s:%s:%s", 
        vote.SessionID.Hex(), vote.QuestionID.Hex(), vote.ParticipantID)
    err = redisDb.GetClient().Set(ctx, participantVoteKey, vote.ID.Hex(), 24*time.Hour).Err()
    if err != nil {
        return fmt.Errorf("failed to cache participant vote mapping: %v", err)
    }

    log.Printf("Vote cached in Redis with multiple access patterns: %s", vote.ID.Hex())
    return nil
}


// Vote count calculation helper methods.
// Helper function to check for duplicate key errors
func isDuplicateKeyError(err error) bool {
    // This would need proper error checking based on MongoDB driver
    // For now, we'll return a simple implementation
    return err != nil && (err.Error() == "duplicate key error")
}
