package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	model "v1.1-fulfiller/models"
)

const (
	connectTimeout           = 5
	connectionStringTemplate = "mongodb+srv://%s:%s@%s"
	database                 = "bitswap"
)

func mongoConnect() (*mongo.Client, context.Context, context.CancelFunc) {
	log.Print("connecting to mongodb")
	username := os.Getenv("MONGODB_USERNAME")
	password := os.Getenv("MONGODB_PASSWORD")
	clusterEndpoint := os.Getenv("MONGODB_ENDPOINT")

	connectionURI := fmt.Sprintf(connectionStringTemplate, username, password, clusterEndpoint)

	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI))
	if err != nil {
		log.Printf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout*time.Second)

	err = client.Connect(ctx)
	if err != nil {
		log.Printf("Failed to connect to cluster: %v", err)
	}

	// Force a connection to verify our connection string
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Printf("Failed to ping cluster: %v", err)
	}

	fmt.Println("Connected to MongoDB!")
	return client, ctx, cancel
}

func CreateOrder(order *model.OrderSchema) error {
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	defer wg.Done()
	order.ID = primitive.NewObjectID()
	_, err := client.Database(database).Collection("orders").InsertOne(ctx, order)
	if err != nil {
		log.Printf("Could not create order: %v", err)
		return err
	}

	return nil
}

func CancelCompleteOrder(orderID string) (err error) {
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	defer wg.Done()
	db := client.Database(database)
	orders := db.Collection("orders")
	update := bson.M{"$set": bson.M{"error": "Cancelled by User", "complete": true, "completeTime": time.Now()}}
	_, err = orders.UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func FulfillOrder(orderID string, cost float64) (err error) {
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	defer wg.Done()
	db := client.Database(database)
	orders := db.Collection("orders")
	users := db.Collection("users")

	err = orders.FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}
	err = users.FindOne(ctx, bson.M{"username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		return err
	}
	var bitcloutBalanceUpdated float64
	var etherBalanceUpdated float64
	//update ether USD price var
	if orderDoc.OrderType == "limit" {
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + (orderDoc.OrderPrice * orderDoc.OrderQuantity)
			etherBalanceUpdated = userDoc.Balance.Ether - (orderDoc.OrderPrice * orderDoc.OrderQuantity / 3000)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - (orderDoc.OrderPrice * orderDoc.OrderQuantity)
			etherBalanceUpdated = userDoc.Balance.Ether + (orderDoc.OrderPrice * orderDoc.OrderQuantity / 3000)
		}
	} else {
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + cost
			etherBalanceUpdated = userDoc.Balance.Ether - (cost / 3000)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - cost
			etherBalanceUpdated = userDoc.Balance.Ether + (cost / 3000)
		}
	}
	if bitcloutBalanceUpdated <= 0 || etherBalanceUpdated <= 0 {
		return errors.New("Insufficient Balance")
	}
	update := bson.M{"$set": bson.M{"orderQuantityProcessed": orderDoc.OrderQuantity, "complete": true, "completeTime": time.Now()}}
	_, err = orders.UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	update = bson.M{"$set": bson.M{"balance.bitclout": bitcloutBalanceUpdated, "balance.ether": etherBalanceUpdated}}
	x, err := users.UpdateOne(ctx, bson.M{"username": orderDoc.Username}, update)
	log.Println("x: ", x)
	if err != nil {
		return err
	}
	return nil
}
func PartialFulfillOrder(orderID string, partialQuantityProcessed float64, cost float64) (err error) {
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	defer wg.Done()
	db := client.Database(database)
	orders := db.Collection("orders")
	users := db.Collection("users")

	err = orders.FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err)
		return err
	}
	err = users.FindOne(ctx, bson.M{"username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		log.Println(err)
		return err
	}
	var bitcloutBalanceUpdated float64
	var etherBalanceUpdated float64
	//update ether USD price var
	if orderDoc.OrderSide == "buy" {
		bitcloutBalanceUpdated = userDoc.Balance.Bitclout + (orderDoc.OrderPrice * partialQuantityProcessed)
		etherBalanceUpdated = userDoc.Balance.Ether - (orderDoc.OrderPrice * partialQuantityProcessed / 3000)
	} else {
		bitcloutBalanceUpdated = userDoc.Balance.Bitclout - (orderDoc.OrderPrice * partialQuantityProcessed)
		etherBalanceUpdated = userDoc.Balance.Ether + (orderDoc.OrderPrice * partialQuantityProcessed / 3000)
	}
	if bitcloutBalanceUpdated <= 0 || etherBalanceUpdated <= 0 {
		log.Println("Insufficient Balance")
		return errors.New("Insufficient Balance")
	}
	update := bson.M{"$set": bson.M{"orderQuantityProcessed": partialQuantityProcessed}}
	_, err = orders.UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		log.Println("Insufficient Balance")
		return err
	}
	update = bson.M{"$set": bson.M{"balance.bitclout": bitcloutBalanceUpdated, "balance.ether": etherBalanceUpdated}}
	_, err = users.UpdateOne(ctx, bson.M{"username": orderDoc.Username}, update)
	if err != nil {
		log.Println("Insufficient Balance")
		return err
	}
	return nil
}
