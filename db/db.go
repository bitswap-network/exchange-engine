package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
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

func MongoConnect() (*mongo.Client, context.CancelFunc) {
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
	return client, cancel
}

func GetUserOrders(ctx context.Context, username string) (orders *[]model.OrderSchema, err error) {
	log.Printf("fetching user orders: %v\n", username)
	var ordersDoc *[]model.OrderSchema

	db := global.Api.Mongo.Database(database)
	orderC := db.Collection("orders")
	cursor, err := orderC.Find(ctx, bson.M{"username": username})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(ctx, &ordersDoc); err != nil {
		log.Println(err)
	}
	log.Println("done fetching orders")
	return ordersDoc, nil
}

func GetUserBalanceFromOrder(ctx context.Context, orderID string) (balance *model.UserBalance, err error) {
	log.Printf("fetching user balance from: %v\n", orderID)
	var userDoc *model.UserSchema
	var orderDoc *model.OrderSchema

	db := global.Api.Mongo.Database(database)
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
	log.Println("done fetching balance")
	return userDoc.Balance, nil
}

func CreateDepthLog(ctx context.Context, depthLog *model.DepthSchema) error {
	log.Println("create depth log")
	_, err := global.Api.Mongo.Database(database).Collection("depths").InsertOne(ctx, depthLog)
	if err != nil {
		log.Printf("Could not create depth log: %v", err)
		return err
	}
	log.Println("done creating depth log")
	return nil
}

func CreateOrder(ctx context.Context, order *model.OrderSchema) error {
	log.Printf("create order: %v\n", order.OrderID)
	order.ID = primitive.NewObjectID()
	_, err := global.Api.Mongo.Database(database).Collection("orders").InsertOne(ctx, order)
	if err != nil {
		log.Printf("Could not create order: %v", err)
		return err
	}
	log.Println("done creating order")
	return nil
}

func CancelCompleteOrder(ctx context.Context, orderID string, errorString string, waitGroup *sync.WaitGroup) error {
	log.Printf("cancel complete: %v\n", orderID)
	defer global.Wg.Done()

	db := global.Api.Mongo.Database(database)
	orders := db.Collection("orders")
	update := bson.M{"$set": bson.M{"error": errorString, "complete": true, "completeTime": time.Now()}}
	_, err := orders.UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func FulfillOrder(ctx context.Context, orderID string, cost float64, waitGroup *sync.WaitGroup) error {
	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
	defer global.Wg.Done()

	db := global.Api.Mongo.Database(database)
	orders := db.Collection("orders")
	users := db.Collection(userCollection())
	//Finding order in database
	err := orders.FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}
	//finding user associated with order
	err = users.FindOne(ctx, bson.M{"username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		return err
	}
	var bitcloutBalanceUpdated, etherBalanceUpdated, bitcloutChange, etherChange float64
	//update ether USD price var
	if orderDoc.OrderType == "limit" {
		if orderDoc.OrderSide == "buy" {
			bitcloutChange = orderDoc.OrderQuantity - orderDoc.OrderQuantity*global.FEE
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + bitcloutChange
			etherChange = (orderDoc.OrderPrice * orderDoc.OrderQuantity / global.ETHUSD)
			etherBalanceUpdated = userDoc.Balance.Ether - etherChange
		} else {
			bitcloutChange = orderDoc.OrderQuantity
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - bitcloutChange
			etherChange = ((orderDoc.OrderPrice * orderDoc.OrderQuantity) - (orderDoc.OrderPrice * orderDoc.OrderQuantity * global.FEE)) / global.ETHUSD
			etherBalanceUpdated = userDoc.Balance.Ether + etherChange
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
	if bitcloutBalanceUpdated < 0 || etherBalanceUpdated < 0 {
		return errors.New("Insufficient Balance")
	}
	update := bson.M{"$set": bson.M{"orderQuantityProcessed": orderDoc.OrderQuantity, "complete": true, "completeTime": time.Now()}}
	_, err = orders.UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	update = bson.M{"$set": bson.M{"balance.bitclout": bitcloutBalanceUpdated, "balance.ether": etherBalanceUpdated}}
	_, err = users.UpdateOne(ctx, bson.M{"username": orderDoc.Username}, update)
	if err != nil {
		return err
	}
	return nil
}

func PartialFulfillOrder(ctx context.Context, orderID string, partialQuantityProcessed float64, cost float64, waitGroup *sync.WaitGroup) (err error) {
	log.Printf("partial fulfill: %v - %v - %v\n", orderID, partialQuantityProcessed, cost)
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
	defer global.Wg.Done()

	db := global.Api.Mongo.Database(database)
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
	var bitcloutBalanceUpdated, etherBalanceUpdated, bitcloutChange, etherChange float64
	if orderDoc.OrderType == "limit" {
		if orderDoc.OrderSide == "buy" {
			bitcloutChange = partialQuantityProcessed - (partialQuantityProcessed * global.FEE)
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + bitcloutChange
			etherChange = (orderDoc.OrderPrice * partialQuantityProcessed) / global.ETHUSD
			etherBalanceUpdated = userDoc.Balance.Ether - etherChange
		} else {
			bitcloutChange = partialQuantityProcessed
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - bitcloutChange
			etherChange = ((orderDoc.OrderPrice * partialQuantityProcessed) - (orderDoc.OrderPrice * partialQuantityProcessed * global.FEE)) / global.ETHUSD
			etherBalanceUpdated = userDoc.Balance.Ether + etherChange
		}
	} else {
		if orderDoc.OrderSide == "buy" {
			bitcloutChange = partialQuantityProcessed * global.FEE
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + bitcloutChange
			etherChange = (cost / global.ETHUSD)
			etherBalanceUpdated = userDoc.Balance.Ether - etherChange
		} else {
			bitcloutChange = partialQuantityProcessed
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - bitcloutChange
			etherChange = (cost - cost*global.FEE) / global.ETHUSD
			etherBalanceUpdated = userDoc.Balance.Ether + etherChange
		}
	}

	if bitcloutBalanceUpdated < 0 || etherBalanceUpdated < 0 {
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
