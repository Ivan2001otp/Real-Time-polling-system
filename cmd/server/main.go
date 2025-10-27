package main

import (
	"RealTimePoll/internal/database"
	"RealTimePoll/internal/routers"
	"RealTimePoll/internal/utils"
	kafkaConfig "RealTimePoll/internal/kafka"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func setUpConfig() {
	
}

func main() {
	
	mongoInstance := database.GetMongoInstance();
	if err := mongoInstance.Init(utils.MONGO_CONNECTION, utils.DB_NAME);err != nil {
		log.Fatal("MongoDB init failed:", err);
	}

	defer mongoInstance.Close();


	redisInstance := database.GetRedisInstance();
	// "redis://localhost:6379"
	if err := redisInstance.Init(utils.REDIS_CONNECTION);err != nil {
		log.Fatal("Redis init failed : ",err);
	}

	defer redisInstance.Close();


	// initialize kafka
	kafkaBrokers := []string{"localhost:29092"} // from docker-compose is picked.
	if err := kafkaConfig.InitKafka(kafkaBrokers); err != nil {
		log.Printf("Kafka init failed: %v (continuing without Kafka)", err);
	}	
	defer kafkaConfig.CloseKafka();

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
	coreRouters := mainRouter.PathPrefix("/api/v1").Subrouter();
	voteRouters := mainRouter.PathPrefix("/api/v1").Subrouter();

	routers.RegisterAuthRoutes(commonRouters);
	routers.RegisterCoreRouters(coreRouters);
	routers.RegisterVotingRouters(voteRouters);

	handler := corsOptions.Handler(mainRouter);
	
	// port = "8080"

	log.Println("Backend listening at port 8080");
	log.Fatal(http.ListenAndServe(":8080",handler));
}

