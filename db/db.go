package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	config "v1.1-fulfiller/config"
	global "v1.1-fulfiller/global"
	model "v1.1-fulfiller/models"
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
}

func GetUserOrders(ctx context.Context, username string) (orders *[]model.OrderSchema, err error) {
	log.Printf("fetching user orders: %v\n", username)
	var ordersDoc *[]model.OrderSchema

	cursor, err := OrderCollection().Find(ctx, bson.M{"username": username})
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

	err = OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	err = UserCollection().FindOne(ctx, bson.M{"username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	log.Println("done fetching balance")
	return userDoc.Balance, nil
}

func CreateDepthLog(ctx context.Context, depthLog *model.DepthSchema) error {
	log.Println("create depth log")
	_, err := DepthCollection().InsertOne(ctx, depthLog)
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
	_, err := OrderCollection().InsertOne(ctx, order)
	if err != nil {
		log.Printf("Could not create order: %v", err)
		return err
	}
	log.Println("done creating order")
	return nil
}

func UpdateOrderPrice(ctx context.Context, orderID string, orderPrice float64, waitGroup *sync.WaitGroup) error {
	log.Printf("update order price: %v\n", orderID)
	defer global.WaitGroup.Done()

	update := bson.M{"$set": bson.M{"orderPrice": orderPrice}}
	_, err := OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)

	if err != nil {
		return err
	}
	return nil
}

func CancelCompleteOrder(ctx context.Context, orderID string, errorString string, waitGroup *sync.WaitGroup) error {

	log.Printf("cancel complete: %v\n", orderID)
	defer global.WaitGroup.Done()

	update := bson.M{"$set": bson.M{"error": errorString, "complete": true, "completeTime": time.Now()}}
	_, err := OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func FulfillOrder(ctx context.Context, orderID string, cost float64, waitGroup *sync.WaitGroup) error {
	ETHUSD := <-global.Exchange.ETHUSD

	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
	defer global.WaitGroup.Done()

	//Finding order in database
	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}
	//finding user associated with order
	err = UserCollection().FindOne(ctx, bson.M{"username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		return err
	}
	var bitcloutBalanceUpdated, etherBalanceUpdated, bitcloutChange, etherChange float64
	//update ether USD price var
	if orderDoc.OrderType == "limit" {
		if orderDoc.OrderSide == "buy" {
			bitcloutChange = orderDoc.OrderQuantity - orderDoc.OrderQuantity*global.Exchange.FEE
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + bitcloutChange
			etherChange = (orderDoc.OrderPrice * orderDoc.OrderQuantity / ETHUSD)
			etherBalanceUpdated = userDoc.Balance.Ether - etherChange
		} else {
			bitcloutChange = orderDoc.OrderQuantity
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - bitcloutChange
			etherChange = ((orderDoc.OrderPrice * orderDoc.OrderQuantity) - (orderDoc.OrderPrice * orderDoc.OrderQuantity * global.Exchange.FEE)) / ETHUSD
			etherBalanceUpdated = userDoc.Balance.Ether + etherChange
		}
	} else {
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + orderDoc.OrderQuantity
			etherBalanceUpdated = userDoc.Balance.Ether - (cost / ETHUSD)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - orderDoc.OrderQuantity
			etherBalanceUpdated = userDoc.Balance.Ether + (cost / ETHUSD)
		}
	}
	if bitcloutBalanceUpdated < 0 || etherBalanceUpdated < 0 {
		return errors.New("Insufficient Balance")
	}
	update := bson.M{"$set": bson.M{"orderQuantityProcessed": orderDoc.OrderQuantity, "complete": true, "completeTime": time.Now()}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	update = bson.M{"$set": bson.M{"balance.bitclout": bitcloutBalanceUpdated, "balance.ether": etherBalanceUpdated}}
	_, err = UserCollection().UpdateOne(ctx, bson.M{"username": orderDoc.Username}, update)
	if err != nil {
		return err
	}
	return nil
}

func PartialFulfillOrder(ctx context.Context, orderID string, partialQuantityProcessed float64, cost float64, waitGroup *sync.WaitGroup) (err error) {
	ETHUSD := <-global.Exchange.ETHUSD
	log.Printf("partial fulfill: %v - %v - %v\n", orderID, partialQuantityProcessed, cost)
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
	defer global.WaitGroup.Done()

	err = OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err)
		return err
	}
	err = UserCollection().FindOne(ctx, bson.M{"username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		log.Println(err)
		return err
	}
	var bitcloutBalanceUpdated, etherBalanceUpdated, bitcloutChange, etherChange float64
	if orderDoc.OrderType == "limit" {
		if orderDoc.OrderSide == "buy" {
			bitcloutChange = partialQuantityProcessed - (partialQuantityProcessed * global.Exchange.FEE)
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + bitcloutChange
			etherChange = (orderDoc.OrderPrice * partialQuantityProcessed) / ETHUSD
			etherBalanceUpdated = userDoc.Balance.Ether - etherChange
		} else {
			bitcloutChange = partialQuantityProcessed
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - bitcloutChange
			etherChange = ((orderDoc.OrderPrice * partialQuantityProcessed) - (orderDoc.OrderPrice * partialQuantityProcessed * global.Exchange.FEE)) / ETHUSD
			etherBalanceUpdated = userDoc.Balance.Ether + etherChange
		}
	} else {
		if orderDoc.OrderSide == "buy" {
			bitcloutChange = partialQuantityProcessed * global.Exchange.FEE
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout + bitcloutChange
			etherChange = (cost / ETHUSD)
			etherBalanceUpdated = userDoc.Balance.Ether - etherChange
		} else {
			bitcloutChange = partialQuantityProcessed
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout - bitcloutChange
			etherChange = (cost - cost*global.Exchange.FEE) / ETHUSD
			etherBalanceUpdated = userDoc.Balance.Ether + etherChange
		}
	}

	if bitcloutBalanceUpdated < 0 || etherBalanceUpdated < 0 {
		log.Println("Insufficient Balance")
		return errors.New("Insufficient Balance")
	}
	update := bson.M{"$set": bson.M{"orderQuantityProcessed": partialQuantityProcessed}}

	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		log.Println("Insufficient Balance")
		return err
	}
	update = bson.M{"$set": bson.M{"balance.bitclout": bitcloutBalanceUpdated, "balance.ether": etherBalanceUpdated}}
	_, err = UserCollection().UpdateOne(ctx, bson.M{"username": orderDoc.Username}, update)
	if err != nil {
		log.Println("Insufficient Balance")
		return err
	}
	return nil
}
