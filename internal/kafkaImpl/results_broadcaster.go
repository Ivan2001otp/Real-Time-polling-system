package kafkaImpl

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"github.com/segmentio/kafka-go"

	"RealTimePoll/internal/realtime"
	"RealTimePoll/internal/utils"
)


// This method consumes voted.updated events and broadcasts via websocket.
func StartResultsBroadcaster(hub *realtime.Hub) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{utils.KAFKA_CONNECTION},
        Topic:   ResultsUpdatedTopic,
        GroupID: utils.KAFKA_RESULTS_BROADCASTER_GROUP, // Different consumer group
        MinBytes: 10e3, // 10KB
        MaxBytes: 10e6, // 10MB
        MaxWait:  1 * time.Second,
	})

	defer reader.Close();

	 log.Println("Results broadcaster started and listening for results.updated events...")

	 for {
		// read messages from kafka
		msg, err := reader.ReadMessage(context.Background());
		if err != nil {
			log.Printf("Error reading results message from Kafka: %v", err)
            continue
		}

		 log.Printf("Received results update: topic=%s partition=%d offset=%d", 
            msg.Topic, msg.Partition, msg.Offset);

		// Process the results event.
		if err := processResultsMessage(hub, msg); err != nil {
            log.Printf("Failed to process results event: %v", err)
            // Continue processing other messages
        }
	 }
}


// processResultsMessage handles a single results.updated Kafka message
func processResultsMessage(hub *realtime.Hub, msg kafka.Message) error {
    var resultsEvent ResultsUpdatedEvent
    if err := json.Unmarshal(msg.Value, &resultsEvent); err != nil {
        return fmt.Errorf("failed to unmarshal results event: %v", err)
    }

    log.Printf("Processing results event: session=%s, question=%s", 
        resultsEvent.SessionID, resultsEvent.QuestionID)

    // Broadcast to WebSocket clients
    return broadcastResultsUpdate(hub, resultsEvent)
}


func broadcastResultsUpdate(hub *realtime.Hub, resultsEvent ResultsUpdatedEvent) error {
	wsMessage := map[string]interface{}{
        "type":       "results_updated",
        "sessionId":  resultsEvent.SessionID,
        "questionId": resultsEvent.QuestionID,
        "results":    resultsEvent.Results,
        "timestamp":  resultsEvent.Timestamp,
        "eventId":    resultsEvent.EventID,
    }

	messageJSON, err := json.Marshal(wsMessage);
	if err != nil {
		return fmt.Errorf("failed to marshal WebSocket message: %v", err);
	}

	// broadcasting to all clients in the session - poll
	hub.BroadcastToSession(resultsEvent.SessionID, messageJSON);

	log.Printf("Results broadcasted via WebSocket: session=%s, question=%s, clients_notified=true", 
        resultsEvent.SessionID, resultsEvent.QuestionID)
    
    return nil
}

func StartAllConsumer(hub *realtime.Hub) {
	go func() {
        log.Println("Starting vote processor consumer...")
        StartVoteConsumer()
    }()

    // Start results broadcaster consumer
    go func() {
        log.Println("Starting results broadcaster consumer...")
        StartResultsBroadcaster(hub)
    }()
}