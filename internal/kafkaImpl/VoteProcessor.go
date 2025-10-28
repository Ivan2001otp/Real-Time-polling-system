package kafkaImpl

import (
	"RealTimePoll/internal/database"
	"RealTimePoll/internal/models"
	"RealTimePoll/internal/utils"
	"RealTimePoll/internal/repository"

	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// This method processes vote and sends to kafka leading to websockets eventually to all clients to display results realtime.
func processVote(voteEvent VoteSubmittedEvent) error {
	// ctx := context.Background();

	 sessionID, err := primitive.ObjectIDFromHex(voteEvent.SessionID)
    if err != nil {
        return fmt.Errorf("invalid session ID: %v", err)
    }

    questionID, err := primitive.ObjectIDFromHex(voteEvent.QuestionID)
    if err != nil {
        return fmt.Errorf("invalid question ID: %v", err)
    }

	_, err = repository.GetSessionByID(sessionID);
	if err != nil {
		return fmt.Errorf("session not found: %v", err);
	}

	// deduplication check
	if err := checkDuplicateVote(sessionID, questionID, voteEvent.ParticipantID); err != nil {
		return fmt.Errorf("duplicate vote: %v", err);
	}

	nowTime := time.Now();
	voteObjectID,_ := primitive.ObjectIDFromHex(voteEvent.VoteID);
	participantObjID, _ :=  primitive.ObjectIDFromHex(voteEvent.ParticipantID);

	vote := models.Vote{
		ID : voteObjectID,
		SessionID: sessionID,
		QuestionID: questionID,
		ParticipantID: participantObjID,
		SelectedOptions: voteEvent.SelectedOptions,
		CreatedAt: nowTime,
		Processed: true,
		ProcessedAt: nowTime,
	}

	if err := repository.SaveVoteToMongo(vote); err != nil {
		return fmt.Errorf("failed to save vote: %v", err);
	}

	// need to implement.
	if err := UpdateRealTimeResults(vote); err != nil {
        log.Printf("Warning: Failed to update real-time results: %v", err)
        // Don't fail the entire process if results update fails
    }

	log.Printf("Successfully processed vote: %s", voteEvent.VoteID)
    return nil
}

// This function checks and prevents duplicate voting.
func checkDuplicateVote(sessionID, questionID primitive.ObjectID, participantID string) error {
	redisDb := database.GetRedisInstance();
	ctx := context.Background();

	voteLockKey := fmt.Sprintf("vote_lock:%s:%s:%s", 
        sessionID.Hex(), questionID.Hex(), participantID)

	acquired, err := redisDb.GetClient().SetNX(ctx, voteLockKey, "1", 24*time.Hour).Result();
	 if err != nil {
        return fmt.Errorf("redis error: %v", err)
    }

    if !acquired {
        return fmt.Errorf("vote already exists for this participant")
    }

	return nil;
}


/*
*************************************************************************************
	Computing stats of voting.
*************************************************************************************
*/


// Updates the results and triggers real time update pipeline
func UpdateRealTimeResults(vote models.Vote) error {
	session, err := repository.GetSessionByID(vote.SessionID);
	if err != nil {
        return fmt.Errorf("failed to get session: %v", err)
    }

	var question models.Question;

	questionFound := false;
	for _, q := range session.Questions {
		if q.ID == vote.QuestionID {
			question = q;
			questionFound = true;
			break;
		}
	}


	if (! questionFound) {
		 return fmt.Errorf("question not found in session");
	}

	// calculate the updated results for this question.
	results, err := calculateQuestionResults(vote.QuestionID, vote.SessionID, question);
	if err != nil {
        return fmt.Errorf("failed to calculate results: %v", err)
    }

	// cache results in redis for faster access.
	if err := cacheResultsInRedis(vote.SessionID, vote.QuestionID, results); err != nil {
		log.Printf("Warning: Failed to cache results in Redis: %v", err)
	}


	// Emit kafka event for real-time processing.
	resultsEvent := ResultsUpdatedEvent{
		EventID : primitive.NewObjectID().Hex(),
		Type : utils.RESULTS_UPDATED_TOPIC,
		SessionID: vote.SessionID.Hex(),
		QuestionID: vote.QuestionID.Hex(),
		Results : results,
		Timestamp: time.Now(),
	}


	// Send to kafka as a result it will lead to real-time broadcast.
	if err := Produce(ResultsUpdatedTopic, vote.SessionID.Hex(), resultsEvent); err != nil {
		return fmt.Errorf("failed to emit results event to Kafka: %v", err);
	}

	log.Printf("Real-time results updated and broadcasted: session=%s, question=%s", 
        vote.SessionID.Hex(), vote.QuestionID.Hex())
    
    return nil;
}


func calculateQuestionResults(questionID, sessionID primitive.ObjectID, question models.Question) (models.QuestionResult, error) {
	// get vote count for each option
	voteCounts , totalVotes, err := getVoteCountsForQuestion(questionID, sessionID)
    if err != nil {
        return models.QuestionResult{}, err
    }

	// calculate option counts and percentages.
	options := make([]models.OptionCount, len(question.Options))
	var percentage float64;
	for i, optionText := range question.Options {
		count := voteCounts[i];
		percentage  = 0.0;
		if (totalVotes > 0){
			percentage = (float64(count)/float64(totalVotes)) * 100
		}

		options[i] = models.OptionCount{
            Index:      i,
            Text:       optionText,
            Count:      count,
            Percentage: percentage,
        }
	}

	// Get unique voters count for this question.
	votersCount, err := getUniqueVotersCount(questionID, sessionID);

	return models.QuestionResult{
        QuestionID:  questionID.Hex(),
        Text:        question.Text,
        Options:     options,
        TotalVotes:  totalVotes,
        VotersCount: votersCount,
    }, nil
}


func getVoteCountsForQuestion(questionID, sessionID primitive.ObjectID) (map[int]int, int , error) {
	mongo := database.GetMongoInstance()
    votesCollection := mongo.GetCollection(utils.VOTES_COLLECTION);
    ctx := context.Background()

	// Aggregate to count votes per option
    pipeline := []bson.M{
        {
            "$match": bson.M{
                "session_id":  sessionID,
                "question_id": questionID,
            },
        },
        {
            "$unwind": "$selected_options",
        },
        {
            "$group": bson.M{
                "_id": "$selected_options",
                "count": bson.M{"$sum": 1},
            },
        },
    }


	cursor, err := votesCollection.Aggregate(ctx, pipeline);
	if err != nil {
		 return nil, 0, fmt.Errorf("aggregation failed: %v", err);
	}

	defer cursor.Close(ctx);


	//initialize votecounts for all options 
	voteCounts := make(map[int]int);
	totalVotes := 0;


	var result struct {
			Option int `bson:"_id"`
			Count int `bson:"count"`
		}

	for cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			continue;
		}

		voteCounts[result.Option] = result.Count
        totalVotes += result.Count
	}

	return voteCounts, totalVotes, nil;

}

// getUniqueVotersCount returns how many unique participants voted on this question
func getUniqueVotersCount(questionID, sessionID primitive.ObjectID) (int, error) {
    mongo := database.GetMongoInstance()
    votesCollection := mongo.GetCollection("votes")
    ctx := context.Background()

    distinctVoters, err := votesCollection.Distinct(ctx, "participant_id", bson.M{
        "session_id":  sessionID,
        "question_id": questionID,
    })

    if err != nil {
        return 0, fmt.Errorf("failed to get distinct voters: %v", err)
    }

    return len(distinctVoters), nil
}


// cacheResultsInRedis stores results in Redis for fast access
func cacheResultsInRedis(sessionID, questionID primitive.ObjectID, results models.QuestionResult) error {
    redis := database.GetRedisInstance()
    ctx := context.Background()

    resultsKey := fmt.Sprintf("results:%s:%s", sessionID.Hex(), questionID.Hex())
    resultsJSON, err := json.Marshal(results)
    if err != nil {
        return fmt.Errorf("failed to marshal results: %v", err)
    }

    // Cache for 1 hour - sessions typically don't last longer
    err = redis.GetClient().Set(ctx, resultsKey, resultsJSON, 1*time.Hour).Err()
    if err != nil {
        return fmt.Errorf("failed to cache results in Redis: %v", err)
    }

    log.Printf("Results cached in Redis: %s", resultsKey)
    return nil
}


// GetCachedResults retrieves results from Redis cache
func GetCachedResults(sessionID, questionID primitive.ObjectID) (*models.QuestionResult, error) {
    redis := database.GetRedisInstance()
    ctx := context.Background()

    resultsKey := fmt.Sprintf("results:%s:%s", sessionID.Hex(), questionID.Hex())
    resultsJSON, err := redis.GetClient().Get(ctx, resultsKey).Result()
    if err != nil {
        return nil, fmt.Errorf("results not found in cache: %v", err)
    }

    var results models.QuestionResult
    if err := json.Unmarshal([]byte(resultsJSON), &results); err != nil {
        return nil, fmt.Errorf("failed to unmarshal cached results: %v", err)
    }

    return &results, nil
}