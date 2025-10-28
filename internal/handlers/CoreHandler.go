package handlers

import (
	handlerUtil "RealTimePoll/internal/handlers/utils"
	KafkaC "RealTimePoll/internal/kafkaImpl"
	"RealTimePoll/internal/models"
	"RealTimePoll/internal/services"
	"RealTimePoll/internal/utils"
	"RealTimePoll/internal/repository"

	"fmt"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateNewPoll(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		utils.ErrorResponse(w, http.StatusBadRequest, "Try out POST request.")
		return
	}

	var newQuestionPoll models.Session
	if err := json.NewDecoder(r.Body).Decode(&newQuestionPoll); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Something wrong with request body format")
		return
	}

	newQuestionPoll.ID = primitive.NewObjectID()
	newQuestionPoll.Status = utils.ACTIVE // by default is set to draft, unless an explicit request is made to active/close the poll.
	newQuestionPoll.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	newQuestionPoll.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	if len(newQuestionPoll.JoinCode) > 0 {
		log.Println("join code is given by user. This should be computer from server. Error")
		utils.ErrorResponse(w, http.StatusBadRequest, "Do not provide join-code in body, it is supposed to be computed from the server.")
		return
	}

	newQuestionPoll.JoinCode = utils.GenerateJoinCode()

	_, err := services.SavePollQuestions(newQuestionPoll)

	if err != nil {
		log.Println("Something wrong while saving question poll.")
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("Question poll is saved successfully. Id -" + newQuestionPoll.ID.Hex())
	utils.JSONResponse(w, http.StatusCreated, map[string]string{
		"message": "Question Poll saved successfully.",
	})
}

func SubmitVoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed. Try POST !")
		return
	}

	var payload models.VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// basic validation
	if err := handlerUtil.ValidateVoteRequest(payload); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// convert Ids to Object ID
	sessionID, err := primitive.ObjectIDFromHex(payload.SessionID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	_, err = primitive.ObjectIDFromHex(payload.QuestionID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	session, err := repository.GetSessionByID(sessionID)
	if err != nil || session.Status != utils.ACTIVE {
		utils.ErrorResponse(w, http.StatusBadRequest, "Session not active or not found")
		return
	}

	//creating vote event for kafka
	voteEvent := KafkaC.VoteSubmittedEvent{
		EventID:         primitive.NewObjectID().Hex(), // Unique event ID
		Type:            utils.VOTES_SUBMITTED_TOPIC,
		VoteID:          primitive.NewObjectID().Hex(), // Will be the actual vote ID
		SessionID:       payload.SessionID,
		QuestionID:      payload.QuestionID,
		ParticipantID:   payload.ParticipantID,
		SelectedOptions: payload.SelectedOptions,
		Timestamp:       time.Now(),
		IPAddress:       handlerUtil.GetIPAddress(r),
		UserAgent:       r.UserAgent(),
	}

	log.Println("Here is the vote event : ")
	log.Println(voteEvent)

	// send to kafka
	if err := KafkaC.ProduceVoteSubmitted(voteEvent); err != nil {
		log.Printf("Failed to send vote to Kafka: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to process vote")
		return
	}

	nowTime, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	// Immediate 202 response
	utils.JSONResponse(w, http.StatusAccepted, map[string]interface{}{
		"message":   "Vote accepted for processing",
		"voteId":    voteEvent.VoteID,
		"sessionId": payload.SessionID,
		"timeStamp": nowTime,
	})
}

func UpdateSessionHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPatch {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed. Try PATCH !")
		return
	}

	sessionId := r.URL.Query().Get("sessionId")
	status := r.URL.Query().Get("status")

	if len(status) == 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "No status provided.")
		return
	}

	switch status {
	case utils.ACTIVE, utils.CLOSED:
		// do nothing - valid status
	default:
		utils.ErrorResponse(w, http.StatusBadRequest, "The provided status is invalid. Only 'closed' and 'active' are allowed.")
		return
	}

	if len(sessionId) == 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "No Session ID provided.")
		return
	}

	SessionHexID, err := primitive.ObjectIDFromHex(sessionId)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid sessionId format")
		return
	}

	err = services.UpdateSessionStatus(SessionHexID, status)
	if err != nil {
		log.Println(err.Error())
		utils.ErrorResponse(w, http.StatusInternalServerError, "Something went wrong while updating status.")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Updated status of session %s to %s", sessionId, status),
	})

}
