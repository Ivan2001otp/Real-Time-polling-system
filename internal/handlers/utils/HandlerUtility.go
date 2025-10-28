package utils

import (
	"fmt"
	"RealTimePoll/internal/models"
	"net/http"
)

func GetIPAddress(r *http.Request) string {
    // Get IP from X-Forwarded-For header if behind proxy
    if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
        return ip
    }
    return r.RemoteAddr
}

func ValidateVoteRequest(req models.VoteRequest) error {
    if req.SessionID == "" {
        return fmt.Errorf("sessionId is required")
    }
    if req.QuestionID == "" {
        return fmt.Errorf("questionId is required")
    }
    if req.ParticipantID == "" {
        return fmt.Errorf("participantId is required")
    }
    if len(req.SelectedOptions) == 0 {
        return fmt.Errorf("selectedOptions cannot be empty")
    }
    return nil
}