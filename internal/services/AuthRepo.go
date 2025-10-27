package services

import (
	"RealTimePoll/internal/database"
	"RealTimePoll/internal/models"
	"RealTimePoll/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	AuthCacheTTL = 48 * time.Hour
)

func FetchOrganizerFromEmail(email string) (*models.User, error) {
	redisDb := database.GetRedisInstance();
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute * 1);
	defer cancel();

	redisKey := utils.USER_KEY_PREFIX + email;
	cachedUser, err := redisDb.GetClient().Get(ctx, redisKey).Result();

	if err == nil {

		var organizer models.User
		if err := json.Unmarshal([]byte(cachedUser), &organizer); err != nil {
			log.Printf("Error unmarshalling cached user : %v", err);

		} else {
			log.Printf("User %s found in Redis cache", email)
            return &organizer, nil
		}
	}


	// fallback to mongo
	 log.Printf("User %s not found in Redis, querying MongoDB", email)
	 mongoDb := database.GetMongoInstance();
	 usersCollection := mongoDb.GetCollection(utils.USERS_COLLECTION);

	 var organizer models.User;
	 err = usersCollection.FindOne(ctx, map[string]string{"email":email}).Decode(&organizer);
	 
	 if err != nil{
		return nil, fmt.Errorf("user not found: %v", err)
	 }

	 // cache the user to redis
	 if err := cacheOrganizerInRedis(&organizer);err != nil {
		log.Printf("Warning: Failed to cache user in Redis: %v", err)
	 }
	 return &organizer, nil;
}

func cacheOrganizerInRedis(organizer *models.User) error {
	redisDb := database.GetRedisInstance();
	ctx, cancel := context.WithTimeout(context.Background(), 60 * time.Second);
	defer cancel();

	userJson, err := json.Marshal(organizer);
	if err != nil {
        return fmt.Errorf("failed to marshal user for Redis: %v", err)
    }

	redisKey := utils.USER_KEY_PREFIX + organizer.Email;
	err = redisDb.GetClient().Set(ctx, redisKey, userJson, AuthCacheTTL).Err();

	 if err != nil {
        return fmt.Errorf("failed to cache user in Redis: %v", err)
    }

    log.Printf("User %s cached in Redis with TTL %v", organizer.Email, AuthCacheTTL);
    return nil
}

func InvalidateUserCache(email string) error {
	redisDb := database.GetRedisInstance();
	
	redisKey := utils.USER_KEY_PREFIX + email;
	err := redisDb.GetClient().Del(context.Background(), redisKey).Err();
	if err != nil {
		return fmt.Errorf("failed to invalidate user cache: %v", err);
	}

	log.Printf("User cache invalidated for: %s", email)
    return nil
}

func SaveOrganizer(user models.User) (error) {
	mongoDb := database.GetMongoInstance();
	usersCollection := mongoDb.GetCollection(utils.USERS_COLLECTION);
	
	ctx , cancel := context.WithTimeout(context.Background(), 1* time.Minute);
	defer cancel();
	
	now,_ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339));
	user.ID = primitive.NewObjectID();
	user.CreatedAt = now
	user.UpdatedAt = now;


	// save the model in mongo
	_, err := usersCollection.InsertOne(ctx, user);
	if err != nil {
		return fmt.Errorf("failed to save user to MongoDB: %v", err);
	}

	  log.Printf("User %s saved to MongoDB", user.Email)

    // Cache in Redis
    if err := cacheOrganizerInRedis(&user); err != nil {
        log.Printf("Warning: Failed to cache user in Redis: %v", err)
        // Don't return error here as MongoDB save was successful
    } else {
		log.Println("Cached the registered user.");
	}
	

    return nil

}