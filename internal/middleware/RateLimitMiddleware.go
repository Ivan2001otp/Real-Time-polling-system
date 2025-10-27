package middleware

import (
	"net/http"
	"log"
	"encoding/json"
	"golang.org/x/time/rate"
)

func RateLimitMiddleware (next func(w http.ResponseWriter, r *http.Request)) http.Handler {
	limiter := rate.NewLimiter(2, 2);//allowing upto event upto r rate, but permits burst of 2 tokens.

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if (! limiter.Allow()) {
				log.Println("Requests overflowed or blocked");
				response := map[string]interface{}{
					"message":"Server is max capacity. Try again later !",
				
				}
				w.WriteHeader(http.StatusTooManyRequests);
				_ = json.NewEncoder(w).Encode(response);

			} else {
				next(w, r);
			}
		})
}