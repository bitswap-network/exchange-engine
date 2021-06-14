package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"exchange-engine/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DbConfig struct {
	Client      *mongo.Client
	Collections CollectionRef
	IsTest      bool
}

type CollectionRef struct {
	Depths string
	Users  string
	Orders string
	Pools  string
}

const (
	connectTimeout           = 5 * time.Second
	connectionStringTemplate = `mongodb+srv://%s:%s@%s/%s?authSource=%%24external&authMechanism=MONGODB-AWS&retryWrites=true&w=majority`
)

var DB = &DbConfig{}

func getCollections() *CollectionRef {
	return &CollectionRef{
		Depths: "depths",
		Users:  "users",
		Orders: "orders",
		Pools:  "pools",
	}
}
func GetDB() *mongo.Database {
	return DB.Client.Database(config.DatabaseConfig.DatabaseName)
}

func DepthCollection() *mongo.Collection {
	return GetDB().Collection(DB.Collections.Depths)
}

func PoolCollections() *mongo.Collection {
	return GetDB().Collection(DB.Collections.Pools)
}

func UserCollection() *mongo.Collection {
	return GetDB().Collection(DB.Collections.Users)
}

func OrderCollection() *mongo.Collection {
	return GetDB().Collection(DB.Collections.Orders)
}

func Close(ctx context.Context) error {
	return DB.Client.Disconnect(ctx)
}

func Setup() {
	log.Println("db setup")
	var err error

	connectionURI := fmt.Sprintf(connectionStringTemplate, config.DatabaseConfig.AWSKey, config.DatabaseConfig.AWSSecret, config.DatabaseConfig.ClusterEndpoint, config.DatabaseConfig.DatabaseName)

	clientOpts := options.Client()

	clientOpts.ApplyURI(connectionURI)
	clientOpts.SetConnectTimeout(connectTimeout)

	DB.Client, err = mongo.NewClient(clientOpts)
	if err != nil {
		log.Panicf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout*time.Second)
	err = DB.Client.Connect(ctx)
	if err != nil {
		log.Panicf("Failed to connect to cluster: %v", err)
	}

	// Force a connection to verify our connection string
	err = DB.Client.Ping(ctx, nil)
	if err != nil {
		log.Panicf("Failed to ping cluster: %v", err)
	}

	fmt.Println("Connected to MongoDB!")
	DB.Collections = *getCollections()
	DB.IsTest = config.IsTest
	defer cancel()
	log.Println("db setup complete")

}
