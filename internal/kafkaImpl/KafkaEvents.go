package kafkaImpl

import (
	"time"
	"RealTimePoll/internal/models"
)

// these are the events of kafka 
type VoteSubmittedEvent struct {
    EventID       string    `json:"eventId"`
    Type          string    `json:"type"` // "vote.submitted"
    VoteID        string    `json:"voteId"`
    SessionID     string    `json:"sessionId"`
    QuestionID    string    `json:"questionId"`
    ParticipantID string    `json:"participantId"`
    SelectedOptions []int   `json:"selectedOptions"`
    Timestamp     time.Time `json:"timestamp"`
    // Add any metadata needed for processing
    IPAddress     string    `json:"ipAddress,omitempty"`
    UserAgent     string    `json:"userAgent,omitempty"`
}

type VoteProcessedEvent struct {
    EventID       string    `json:"eventId"`
    Type          string    `json:"type"` // "vote.processed"
    VoteID        string    `json:"voteId"`
    SessionID     string    `json:"sessionId"`
    QuestionID    string    `json:"questionId"`
    Success       bool      `json:"success"`
    Error         string    `json:"error,omitempty"`
    Timestamp     time.Time `json:"timestamp"`
}

type ResultsUpdatedEvent struct {
    EventID   string         `json:"eventId"`
    Type      string         `json:"type"` // "results.updated"
    SessionID string         `json:"sessionId"`
    QuestionID string        `json:"questionId"`
    Results   models.QuestionResult `json:"results"`
    Timestamp time.Time      `json:"timestamp"`
}