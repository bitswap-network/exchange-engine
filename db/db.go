package db

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
	global "v1.1-fulfiller/global"
	model "v1.1-fulfiller/models"
)

const (
	connectTimeout           = 5
	connectionStringTemplate = "mongodb+srv://%s:%s@%s"
	database                 = "bitswap"
)

func userCollection() string {
	if os.Getenv("ENV_MODE") == "release" {
		return "users"
	} else {
		return "test_users"
	}
}

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

func GetUserBalanceFromOrder(orderID string) (balance *model.UserBalance, err error) {
	log.Printf("user balance: %v\n", orderID)
	var userDoc *model.UserSchema
	var orderDoc *model.OrderSchema
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database(database)
	orders := db.Collection("orders")
	users := db.Collection(userCollection())
	err = orders.FindOne(ctx, bson.M{"orderID": orderID}).Decode(&userDoc)
	if err != nil {
		return nil, err
	}
	err = users.FindOne(ctx, bson.M{"username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		return nil, err
	}
	return userDoc.Balance, nil
}

func CreateOrder(order *model.OrderSchema) error {
	log.Printf("create order: %v\n", order.OrderID)
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	order.ID = primitive.NewObjectID()
	_, err := client.Database(database).Collection("orders").InsertOne(ctx, order)
	if err != nil {
		log.Printf("Could not create order: %v", err)
		return err
	}
	return nil
}

func CancelCompleteOrder(orderID string, errorString string) error {
	log.Printf("cancel complete: %v\n", orderID)
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	defer global.Wg.Done()
	db := client.Database(database)
	orders := db.Collection("orders")
	update := bson.M{"$set": bson.M{"error": errorString, "complete": true, "completeTime": time.Now()}}
	_, err := orders.UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func FulfillOrder(orderID string, cost float64) error {
	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	defer global.Wg.Done()
	db := client.Database(database)
	orders := db.Collection("orders")
	users := db.Collection(userCollection())
	//Finding order in database
	err := orders.FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}
	log.Println(orderDoc)
	//finding user associated with order
	err = users.FindOne(ctx, bson.M{"username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		return err
	}
	var bitcloutBalanceUpdated float64
	var etherBalanceUpdated float64
	//update ether USD price var
	if orderDoc.OrderType == "limit" {
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + orderDoc.OrderQuantity
			etherBalanceUpdated = userDoc.Balance.Ether - (orderDoc.OrderPrice * orderDoc.OrderQuantity / global.ETHUSD)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - orderDoc.OrderQuantity
			etherBalanceUpdated = userDoc.Balance.Ether + (orderDoc.OrderPrice * orderDoc.OrderQuantity / global.ETHUSD)
		}
	} else {
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + orderDoc.OrderQuantity
			etherBalanceUpdated = userDoc.Balance.Ether - (cost / global.ETHUSD)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - orderDoc.OrderQuantity
			etherBalanceUpdated = userDoc.Balance.Ether + (cost / global.ETHUSD)
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
	log.Printf("partial fulfill: %v - %v - %v\n", orderID, partialQuantityProcessed, cost)
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
	client, ctx, cancel := mongoConnect()
	defer cancel()
	defer client.Disconnect(ctx)
	defer global.Wg.Done()
	db := client.Database(database)
	orders := db.Collection("orders")
	users := db.Collection(userCollection())

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
	var bitcloutBalanceUpdated, etherBalanceUpdated float64
	if orderDoc.OrderType == "limit" {
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + partialQuantityProcessed
			etherBalanceUpdated = userDoc.Balance.Ether - (orderDoc.OrderPrice * partialQuantityProcessed / global.ETHUSD)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - partialQuantityProcessed
			etherBalanceUpdated = userDoc.Balance.Ether + (orderDoc.OrderPrice * partialQuantityProcessed / global.ETHUSD)
		}
	} else {
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + partialQuantityProcessed
			etherBalanceUpdated = userDoc.Balance.Ether - (cost / global.ETHUSD)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - partialQuantityProcessed
			etherBalanceUpdated = userDoc.Balance.Ether + (cost / global.ETHUSD)
		}
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
