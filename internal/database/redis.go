package database

import (
	"sync"
	"log"
	"fmt"
	"context"
	"github.com/redis/go-redis/v9"
)

type redisDB struct {
	client *redis.Client;
}

var (
	redisInstance *redisDB
	redisOnce sync.Once
	redisMu sync.RWMutex
)

func GetRedisInstance() *redisDB{
	redisOnce.Do(func() {
		redisInstance = &redisDB{}
	})

	return redisInstance;
}

func (r *redisDB)Init(redisURL string) error {
	redisMu.Lock();
	defer redisMu.Unlock();

	if r.client != nil {
		if err := r.client.Ping(context.Background()).Err(); err != nil {
			log.Println("Redis already connected")
            return nil
		}
	}

	opts, err:= redis.ParseURL(redisURL)
	if err != nil {
		log.Println("Given redis url is invalid : ",redisURL);
		return fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	r.client  = redis.NewClient(opts);

	ctx := context.Background();
	err = r.client.Ping(ctx).Err();
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis successfully")
    return nil
}

func (r *redisDB) GetClient() *redis.Client {
    redisMu.RLock()
    defer redisMu.RUnlock()
    return r.client
}

func (r *redisDB) Close() {
    redisMu.Lock()
    defer redisMu.Unlock()

    if r.client != nil {
        r.client.Close()
        log.Println("Redis connection closed")
    }
}

func (r *redisDB) IsConnected() bool {
    redisMu.RLock()
    defer redisMu.RUnlock()

    if r.client == nil {
        return false
    }

    return r.client.Ping(context.Background()).Err() == nil
}