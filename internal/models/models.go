package models

import ("go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	ID primitive.ObjectID `bson:"_id" json:"id"`
	Email string `bson:"email" json:"email"`
	PasswordHash string `bson:"password_hash" json:"passwordHash"`
	Name string `bson:"name" json:"name"`
	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time `bson:"updated_at" json:"updatedAt"`
}


type Session struct {
	ID primitive.ObjectID `bson:"_id" json:"id"`
	OrganizerId primitive.ObjectID `bson:"organizer_id" json:"organizerId"`
	JoinCode string `bson:"join_code" json:"joinCode"`
	Title string `bson:"title" json:"title"`
	Status string `bson:"status" json:"status"`//active,closed,drafted
	Questions []Question `bson:"questions" json:"questions"`
	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time `bson:"updated_at" json:"updatedAt"` 
}

type Question struct {
	ID primitive.ObjectID `bson:"id,omitempty" json:"id"`
	Text string `bson:"text" json:"text"`
	Options []string `bson:"options" json:"options"`
	Type string `bson:"type" json:"type"`//single or multiple
}

type Vote struct {
	ID primitive.ObjectID `bson:"_id" json:"id"`
	SessionID primitive.ObjectID `bson:"session_id" json:"sessionId"`
	QuestionID primitive.ObjectID `bson:"question_id" json:"questionId"`
	ParticipantID primitive.ObjectID `bson:"participant_id" json:"participantId"`
	SelectedOptions []int `bson:"selected_options" json:"selectedOptions"`
	CreatedAt time.Time `bson:"created_at" json:"createdAt"`
	Processed bool `bson:"processed" json:"processed"`
	ProcessedAt time.Time `bson:"processed_at" json:"processedAt,omitempty"`
}



// important models to represent stats across every question poll result.
type QuestionResult struct {
	QuestionID string `json:"questionId"`
	Text string `json:"text"`
	Options []OptionCount `json:"options"`
	TotalVotes int `json:"totalVotes"`
	VotersCount int `json:"votersCount"`
}

type OptionCount struct {
	Index int `json:"index"`
	Text string `json:"text"`
	Count int `json:"count"`
	Percentage float64 `json:"percentage"`
}

type SessionResults struct {
    SessionID    string                  `json:"sessionId"`
    Title        string                  `json:"title"`
    Status       string                  `json:"status"`
    TotalVotes   int                     `json:"totalVotes"`
    Participants int                     `json:"participants"` // Unique participants
    Questions    map[string]QuestionResult `json:"questions"` // questionID -> results
    LastUpdated  time.Time               `json:"lastUpdated"`
}


// request body to submit vote.
type VoteRequest struct {
    SessionID     string   `json:"sessionId"`
    QuestionID    string   `json:"questionId"`
    ParticipantID string   `json:"participantId"` // unique id generated from frontend
    SelectedOptions []int  `json:"selectedOptions"`
}