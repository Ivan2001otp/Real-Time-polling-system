package handlers

import (
	"RealTimePoll/internal/models"
	"RealTimePoll/internal/services"
	"RealTimePoll/internal/utils"

	"encoding/json"
	"log"
	"net/http"
	"time"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateNewPoll(w http.ResponseWriter, r *http.Request) {

	if (r.Method != http.MethodPost) {
		utils.ErrorResponse(w, http.StatusBadRequest, "Try out POST request.");
		return;
	}

	var newQuestionPoll models.Session;
	if err := json.NewDecoder(r.Body).Decode(&newQuestionPoll);err!= nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Something wrong with request body format");
		return;
	}

	newQuestionPoll.ID = primitive.NewObjectID();
	newQuestionPoll.Status = utils.DRAFT; // by default is set to draft, unless an explicit request is made to active/close the poll.
	newQuestionPoll.CreatedAt,_ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339));
	newQuestionPoll.UpdatedAt,_ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339));

	if len(newQuestionPoll.JoinCode) > 0 {
		log.Println("join code is given by user. This should be computer from server. Error");
		utils.ErrorResponse(w, http.StatusBadRequest, "Do not provide join-code in body, it is supposed to be computed from the server.");
		return;
	} 

	newQuestionPoll.JoinCode = utils.GenerateJoinCode();
	

	id, err := services.SavePollQuestions(newQuestionPoll);

	if err != nil {
		log.Println("Something wrong while saving question poll.");
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error());
		return;
	}

	log.Printf("Question poll with Id %s is saved successfully.", id);
	utils.JSONResponse(w, http.StatusCreated, map[string]string{
		"message":fmt.Sprintf("Question Poll saved successfully . Question-poll-Id is %s",id),
	});
}
