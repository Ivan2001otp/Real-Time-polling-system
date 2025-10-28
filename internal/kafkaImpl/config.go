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


const (
	VotesSubmittedTopic = "votes.submitted"
	VotesProcessedTopic = "votes.processed"
	ResultsUpdatedTopic = "votes.updated"
)


type KafkaConfig struct {
	Brokers []string;
}

var (
	writer *kafka.Writer;
)


func InitKafka(brokers []string) error {
	writer = &kafka.Writer{
		Addr: kafka.TCP(brokers...),
		Balancer: &kafka.Hash{}, // hash partition by key
		RequiredAcks: kafka.RequireOne,
		Async: false, // set to true for higher throughput(risk of data loss)
	}

	 log.Printf("Kafka initialized with brokers: %v", brokers)
    return nil
}


func CloseKafka() {
	if writer != nil {
		writer.Close();
	}
}

func Produce(topic string, key string, value interface{}) error {
    // We'll implement this next - serializing the value to JSON
	if writer == nil {
		 return fmt.Errorf("kafka writer not initialized");
	}

	//serialize the value to json
	jsonValue, err := json.Marshal(value);
	if err != nil {
		  return fmt.Errorf("failed to marshal message: %v", err);
	}

	message := kafka.Message{
        Topic: topic,
        Key:   []byte(key), // Partition by sessionID for ordering
        Value: jsonValue,
        Time:  time.Now(),
    }

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second);
	defer cancel();


	err = writer.WriteMessages(ctx, message);
	if err != nil {
        return fmt.Errorf("failed to write message to Kafka: %v", err)
    }

    
    log.Printf("Produced message to topic %s with key %s", topic, key)
    return nil
}


func ProduceVoteSubmitted(voteEvent interface{}) error {
	var sessionID string

	switch v := voteEvent.(type){
	case map[string]interface{}:
		if id, ok := v["sessionId"].(string); ok {
			sessionID = id;
		}


	default:
			sessionID = "default";//using default-key later will improve it.
	}


	return Produce(utils.VOTES_SUBMITTED_TOPIC, sessionID, voteEvent);
}