package routers

import (
	"RealTimePoll/internal/handlers"
	"RealTimePoll/internal/utils"
	"RealTimePoll/internal/middleware"
	"RealTimePoll/internal/realtime"
	"RealTimePoll/pkg/jwt"
	"net/http"

	"github.com/gorilla/mux"
)

func RegisterCoreRouters(apiRouter *mux.Router) {
	apiRouter.Use(func(next http.Handler) http.Handler {
		return middleware.RateLimitMiddleware(next.ServeHTTP)
	});

	//token middleware
	apiRouter.Use(func(h http.Handler) http.Handler {
		return jwt.Middleware(h.ServeHTTP);
	})

	apiRouter.HandleFunc("/sessions", handlers.CreateNewPoll).Methods("POST");
}


func RegisterVotingRouters(apiRouter *mux.Router) {
	

	apiRouter.HandleFunc("/votes", handlers.SubmitVoteHandler).Methods("POST");
	apiRouter.HandleFunc("/status", handlers.UpdateSessionHandler).Methods("PATCH");
}

func RegisterWebsocketRoutes(apiRouter *mux.Router, hub *realtime.Hub) {
	
	apiRouter.Use(func(h http.Handler) http.Handler {
		return jwt.Middleware(h.ServeHTTP);
	});

	apiRouter.HandleFunc("/ws", hub.ServeWebSocket);

	apiRouter.HandleFunc("/api/v1/ws/stats", func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("sessionId");
		if sessionID == "" {
			utils.ErrorResponse(w, http.StatusBadRequest, "sessionId is absent");
			return;
		}


		stats := hub.GetSessionStats(sessionID);
		utils.JSONResponse(w, http.StatusOK,stats);
	})
}