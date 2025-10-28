package handlers

import (
	"RealTimePoll/internal/models"
	"RealTimePoll/internal/services"
	"RealTimePoll/internal/utils"
	"RealTimePoll/pkg/jwt"

	"encoding/json"
	"net/http"
	"fmt"
)

func RegisterOrganizerHandler(w http.ResponseWriter, r *http.Request) {

	if (r.Method != http.MethodPost) {
		utils.ErrorResponse(w, http.StatusBadRequest, "Try out POST request.");
		return;
	}

	var organizer models.User;
	err := json.NewDecoder(r.Body).Decode(&organizer);
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Something wrong with request body format");
		return;
	}

	


	// check whether user exists with the provided email
	fetchedUser, err := services.FetchOrganizerFromEmail(organizer.Email);
	if fetchedUser != nil {
		utils.ErrorResponse(w, http.StatusConflict, "User Already exists for the given email - "+organizer.Email);
		return;
	}

	// hash password
	hashedPassword, err := utils.HashPassword(organizer.PasswordHash);
	 if err != nil {
        utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to process password")
        return
    }
	organizer.PasswordHash = hashedPassword;

	// generate jwt key
	token, err := jwt.GenerateToken(organizer.ID.Hex(), organizer.Email);
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to generate authtoken.")
        return
	}


	// save user
	if err := services.SaveOrganizer(organizer); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
        return
	}


	utils.JSONResponse(w, http.StatusCreated, map[string]string{
		"message":fmt.Sprintf("User with  Registered successfully %s", organizer.Email),
		"token" : token,
	});
	
}

func LoginOrganizerHandler(w http.ResponseWriter, r *http.Request) {

	if (r.Method != http.MethodGet) {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "The current method is not allowed.Try GET method.")
		return;
	}

	var organizer models.User
	if err := json.NewDecoder(r.Body).Decode(&organizer); err != nil {
        utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
        return
    }

	userFromDb, err := services.FetchOrganizerFromEmail(organizer.Email);
	 if err != nil {
        utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
        return
    } 

	// verify password
	if (! utils.CheckPasswordHash(organizer.PasswordHash, userFromDb.PasswordHash)){
		utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid credentials (password)")
        return
	}

	token, err := jwt.GenerateToken(userFromDb.ID.Hex(), userFromDb.Email)
    if err != nil {
        utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
        return
    }


	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"token":token,
		"user":userFromDb,
	});
}

