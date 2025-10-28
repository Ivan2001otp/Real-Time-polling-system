package kafkaImpl

import (
	"RealTimePoll/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// This consumes votes from kafka
func StartVoteConsumer() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers : []string{utils.KAFKA_CONNECTION},
		Topic : utils.VOTES_SUBMITTED_TOPIC,
		GroupID: utils.KAFKA_GROUP_ID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait: 1 * time.Second,
	});

	defer reader.Close();

	log.Println("Vote Consumer started...");

	for {
		msg, err := reader.ReadMessage(context.Background());
		if err != nil {
			log.Printf("Error reading message: %v", err)
            continue
		}

		log.Printf("Received vote message: topic=%s partition=%d offset=%d", 
            msg.Topic, msg.Partition, msg.Offset);

		// process the vote message.
		if err := processVoteMessageWithRetry(msg, 3); err != nil {
			log.Printf("Failed to process vote after retries: %v", err)
			// optionally send to dead letter qeueue here
		}
	}
}



func processVoteMessageWithRetry(msg kafka.Message, maxRetries int) error {
	var err error;
	for attempt :=0 ;attempt <= maxRetries; attempt++ {
		err = processVoteMessage(msg);

		if err == nil {
			// break out of here !
			return nil;
		}

		if attempt == maxRetries {
			break;
		}

		if isNonRetriableError(err) {
			log.Printf("Non-retriable error, skipping retry: %v", err)
            return err
		}

		backoff := calculateBackoff(attempt);
		log.Printf("Vote processing failed (attempt %d/%d), retrying in %v: %v", 
            attempt+1, maxRetries, backoff, err)
        
        time.Sleep(backoff)
 	}

	return fmt.Errorf("failed to process vote after %d attempts: %v", maxRetries, err);
}

func calculateBackoff(attempt int) time.Duration {
	baseDelay := time.Second;
	maxDelay := 30 * time.Second;

	// exponential backoff : 1s, 2s, 4s, 8s,...
	delay := baseDelay * time.Duration(1 << uint(attempt));


	// add jitter(+-20%) to avoid thundering herd.
	  // Cap at max delay
    if delay > maxDelay {
        delay = maxDelay
    }

    return delay
}


// isNonRetriableError checks if error should not be retried
func isNonRetriableError(err error) bool {
    // Don't retry on these errors:
    return err != nil && (
        err.Error() == "duplicate vote" ||
        err.Error() == "session is not active" ||
        err.Error() == "session not found" ||
        err.Error() == "invalid session ID" ||
        err.Error() == "invalid question ID")
}

func processVoteMessage(msg kafka.Message) error {
	var voteEvent VoteSubmittedEvent;
	if err := json.Unmarshal(msg.Value, &voteEvent); err != nil {
		return fmt.Errorf("failed to unmarshal vote event: %v", err);
	}

	log.Printf("Processing vote: eventId=%s, sessionId=%s", voteEvent.EventID, voteEvent.SessionID);

	return processVote(voteEvent);
}