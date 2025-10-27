package routers

import (
	"RealTimePoll/internal/handlers"
	"RealTimePoll/internal/middleware"
	"net/http"

	"github.com/gorilla/mux"
)

func RegisterAuthRoutes(apiRouter *mux.Router) {
	apiRouter.Use(func(next http.Handler) http.Handler {
		return middleware.RateLimitMiddleware(next.ServeHTTP)
	})

	apiRouter.HandleFunc("/login", handlers.LoginOrganizerHandler).Methods("GET")
	apiRouter.HandleFunc("/register", handlers.RegisterOrganizerHandler).Methods("POST")
}
