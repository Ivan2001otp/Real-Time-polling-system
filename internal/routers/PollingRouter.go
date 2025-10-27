package routers

import (
	"github.com/gorilla/mux"
	"RealTimePoll/internal/middleware"
	"RealTimePoll/internal/handlers"
	"net/http"
)

func RegisterCoreRouters(apiRouter *mux.Router) {
	apiRouter.Use(func(next http.Handler) http.Handler {
		return middleware.RateLimitMiddleware(next.ServeHTTP)
	});

	//token middleware

	apiRouter.HandleFunc("/sessions", handlers.CreateNewPoll);
}