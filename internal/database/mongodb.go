package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoDB struct {
    client   *mongo.Client
    database *mongo.Database
}

var (
    instance *mongoDB
    once     sync.Once
    mu       sync.RWMutex
)

func GetMongoInstance() *mongoDB{
	once.Do(func(){
		instance = &mongoDB{}
	})
	return instance;
}


// initialization of mongodb connection.
func (m *mongoDB) Init(connectionString, dbName string) error {
	mu.Lock();
	defer mu.Unlock();


	if (m.client != nil) {
		if err := m.client.Ping(context.Background(), nil); err != nil {
			log.Println("MongoDB already connected");
			return nil;
		}
	}


	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second);
	defer cancel();

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString)) ;
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}


	// ping the db
	err = client.Ping(ctx, nil)
	if err != nil {
        return fmt.Errorf("failed to ping MongoDB: %v", err)
    }

	 m.client = client
    m.database = client.Database(dbName)

    log.Println("Connected to MongoDB successfully")
    return nil
}


// get-collection in a thread safe way
func (m *mongoDB) GetCollection(collectionName string) *mongo.Collection{
	mu.RLock();
	defer mu.RUnlock();

	if (m.database == nil) {
		log.Fatal("MongoDB not initialized. Call Init() first.");
	}

	return m.database.Collection(collectionName);
}


func (m *mongoDB) GetClient() *mongo.Client {
	mu.RLock();
	defer mu.RUnlock();
	return m.client;
}


func (m *mongoDB) Close() {
	mu.Lock()
	defer mu.Unlock();

	if (m.client != nil) {
		ctx, cancel := context.WithTimeout(context.Background() , 10 * time.Second)
		defer cancel();
		m.client.Disconnect(ctx);
		log.Println("MongoDB connection closed")
	}
}

func (m *mongoDB) IsConnected() bool {
	mu.RLock();
	defer mu.RUnlock();

	if (m.client == nil) {
		return false;
	}


	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel();

	return m.client.Ping(ctx, nil) == nil;
}