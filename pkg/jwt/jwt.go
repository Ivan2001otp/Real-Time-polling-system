package jwt

import (
	"context"
	"errors"
	"net/http"
	"time"
	"github.com/dgrijalva/jwt-go"

	"RealTimePoll/internal/utils"
)

var jwtSecretKey = []byte("this-is-jwt-secret-key") // this is only for project purpose.
// Instead we can add SHA generated fingerprint key

type Claims struct {
	OrganizerID string `json:"organizer_id"`
	Email       string `json:"email"`
	jwt.StandardClaims
}

func GenerateToken(organizerId, email string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		OrganizerID: organizerId,
		Email:       email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			Issuer:    utils.JWT_CLAIM_ISSUER,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecretKey)
}

// validate the jwt token
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// jwt middleware
func Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get(utils.AUTHORIZATION_HEADER);

		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized);
			return;
		}

		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
            http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
            return
        }

		tokenString := authHeader[7:]
        claims, err := ValidateToken(tokenString)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }


		ctx := context.WithValue(r.Context(), "organizerClaims", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
	}
}
