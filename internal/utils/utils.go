package utils

import (
	"crypto/rand"
	"encoding/json"
	"encoding/hex"
	"net/http"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost);
	return string(bytes), err;
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil;
}

// generates joincode.
func GenerateJoinCode() string {
	bytes := make([]byte, 3);
	rand.Read(bytes);
	return hex.EncodeToString(bytes);
}


func JSONResponse(w http.ResponseWriter,  status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json");
	w.WriteHeader(status);
	json.NewEncoder(w).Encode(data);
}

func ErrorResponse(w http.ResponseWriter, status int , message string) {
	JSONResponse(w, status, map[string]string{"error":message});
}