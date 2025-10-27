package main

import (
	"RealTimePoll/internal/database"
	"RealTimePoll/internal/routers"
	"RealTimePoll/internal/utils"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func setUpConfig() {
	
}

func main() {
	
	mongoInstance := database.GetMongoInstance();
	if err := mongoInstance.Init("mongodb://localhost:27017", "polling");err != nil {
		log.Fatal("MongoDB init failed:", err);
	}

	defer mongoInstance.Close();


	redisInstance := database.GetRedisInstance();
	if err := redisInstance.Init("redis://localhost:6379");err != nil {
		log.Fatal("Redis init failed : ",err);
	}

	defer redisInstance.Close();

	// cors setup
	corsOptions := cors.New(cors.Options{
		AllowedMethods :[]string{"GET", "DELETE", "POST", "OPTIONS", "PUT", "PATCH"},
		AllowCredentials : true,
		AllowedHeaders:[]string{"Authorization", "Content-Type"},
		
	})


	// setup routes
	mainRouter := mux.NewRouter();

	mainRouter.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type","application/json");
		w.WriteHeader(http.StatusNotFound);

		utils.ErrorResponse(w, http.StatusNotFound, "The API endpoint you trying to reach does not exist.Make sure you are trying out the right one.");
	})


	commonRouters := mainRouter.PathPrefix("/api/v1").Subrouter();
	routers.RegisterAuthRoutes(commonRouters);

	handler := corsOptions.Handler(mainRouter);
	
	// port = "8080"

	log.Println("Backend listening at port 8080");
	log.Fatal(http.ListenAndServe(":8080",handler));
}

