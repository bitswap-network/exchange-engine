package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"v1.1-fulfiller/config"
	"v1.1-fulfiller/global"
	"v1.1-fulfiller/models"
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

/*
Retrieves and populates a user doc from the database given the username

Arguments:
	ctx - The context from which the function is being called
	username - The username of the user you are searching for

Returns:
	The userDoc retrieved from mongo
	An error if:
		(most likely) the user could not be found
		mongo connection failed or something went wrong.
*/
func GetUserDoc(ctx context.Context, username string) (*models.UserSchema, error) {
	var userDoc *models.UserSchema
	err := UserCollection().FindOne(ctx, bson.M{"bitclout.username": username}).Decode(&userDoc)
	if err != nil {
		log.Printf("Could not find user: %v\n"+err.Error(), username)
		return nil, err
	}
	return userDoc, nil
}

/*
Updates a user's bitclout and ether balance

Arguments:
	ctx - The context from which the function is being called
	username - The username of the user you are searching for
	bitcloutChange - The change in bitClout quantity
	etherChange - The change in ether quantity

Returns:
	An error if:
		(most likely) The user's balance becomes invalid (below 0)
		mongo connection failed or something went wrong.
*/
func UpdateUserBalance(ctx context.Context, username string, bitcloutChange, etherChange float64) error {
	update := bson.M{"$inc": bson.M{"balance.bitclout": bitcloutChange, "balance.ether": etherChange}}
	_, err := UserCollection().UpdateOne(ctx, bson.M{"bitclout.username": username}, update)
	if err != nil {
		return err
	}
	return nil
}

/*
Retrieves user orders from the database given the username

Arguments:
	ctx - The context from which the function is being called
	username - The username of the user you are searching for

Returns:
	An array of orders
	An error if:
		(most likely) the user could not be found
		mongo connection failed or something went wrong.
*/
func GetUserOrders(ctx context.Context, username string) ([]models.OrderSchema, error) {
	var ordersArray []models.OrderSchema
	cursor, err := OrderCollection().Find(ctx, bson.M{"username": username, "complete": false})
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		//Create a value into which the single document can be decoded
		var elem models.OrderSchema
		err := cursor.Decode(&elem)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		ordersArray = append(ordersArray, elem)
	}
	return ordersArray, nil
}

/*
Retrieves a user's balance from the database given the username

Arguments:
	ctx - The context from which the function is being called
	username - The username of the user you are searching for

Returns:
	The user's balance
	An error if:
		(most likely) the user could not be found
		mongo connection failed or something went wrong.
*/
func GetUserBalance(ctx context.Context, username string) (balance *models.UserBalance, err error) {
	userDoc, err := GetUserDoc(ctx, username)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return userDoc.Balance, nil
}

func CreateDepthLog(ctx context.Context, depthLog *models.DepthSchema) error {
	log.Println("create depth log")
	_, err := DepthCollection().InsertOne(ctx, depthLog)
	if err != nil {
		log.Printf("Could not create depth log: %v", err)
		return err
	}
	log.Println("done creating depth log")
	return nil
}

/*
Pushes an order to the database

Arguments:
	ctx - The context from which the function is being called
	order - The order to push

Returns:
	An error if:
		(most likely) the error could not be pushed
		mongo connection failed or something went wrong.
*/
func CreateOrder(ctx context.Context, order *models.OrderSchema) error {
	order.ID = primitive.NewObjectID()
	_, err := OrderCollection().InsertOne(ctx, order)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

/*
Updates an order's `orderQuantityProcessed` and `orderPrice` fields in the database.
This is most likely used to update orders after fulfilling partial buy or sell orders.

Arguments:
	ctx - The context from which the function is being called
	order - The order to update

Returns:
	An error if:
		(most likely) the error could not be updated
		mongo connection failed or something went wrong.
*/
func UpdateOrder(ctx context.Context, order *models.OrderSchema) error {
	log.Printf("updating order: %v\n", order.OrderID)
	filter := bson.M{"orderID": order.OrderID}
	update := bson.M{"$set": bson.M{
		"orderQuantityProcessed": order.OrderQuantity,
		"orderPrice":             order.OrderPrice,
	}}
	_, err := OrderCollection().UpdateOne(ctx, filter, update)

	if err != nil {
		log.Println(err.Error())
		return err
	}
	log.Println("done creating order")
	return nil
}

/*
Cancels an order given its ID.

Arguments:
	ctx - The context from which the function is being called
	orderID - The ID of the order to cancel
	errorString - A reason for cancelling the order

Returns:
	An error if:
		(most likely) the order could not be cancelled
		mongo connection failed or something went wrong.
*/
func CancelCompleteOrder(ctx context.Context, orderID string, errorString string) error {
	log.Printf("cancel complete: %v\n", orderID)

	update := bson.M{"$set": bson.M{"error": errorString, "complete": true, "completeTime": time.Now().UTC()}}
	_, err := OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

/*
Completes a limit buy or sell order

Arguments:
	ctx - The context from which the function is being called
	orderID - The ID of the order to complete
	execPrice - The price at which the limit order is executed (sold or bought)

Returns:
	An error if:
		(most likely) the order could not be completed
		mongo connection failed or something went wrong.
*/
func CompleteLimitOrder(ctx context.Context, orderID string, execPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD

	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *models.OrderSchema

	//Finding order in database
	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}

	var bitcloutChange, etherChange float64
	//update ether USD price var
	if orderDoc.OrderSide == "buy" {
		bitcloutChange = (orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) - ((orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) * global.Exchange.FEE)
		etherChange = -(execPrice / ETHUSD)
	} else {
		bitcloutChange = -(orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed)
		etherChange = (execPrice - (execPrice * global.Exchange.FEE)) / ETHUSD
	}

	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update := bson.M{"$set": bson.M{
		"orderQuantityProcessed": orderDoc.OrderQuantity,
		"complete":               true,
		"completeTime":           time.Now().UTC(),
		"execPrice":              (execPrice / (orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed)),
	}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}

	return nil
}

/*
Completes a partial limit order

Arguments:
	ctx - The context from which the function is being called
	orderID - The ID of the order to complete
	partialQuantityProcessed - The quantity of the total order that has been processed
	execPrice - The price at which the limit order is executed (sold or bought)

Returns:
	An error if:
		(most likely) the limit order could not be partially fulfilled
		mongo connection failed or something went wrong.
*/
func PartialLimitOrder(ctx context.Context, orderID string, partialQuantityProcessed float64, execPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("partial fulfill: %v - %v\n", orderID, partialQuantityProcessed)
	var orderDoc *models.OrderSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err)
		return err
	}

	var bitcloutChange, etherChange float64
	if orderDoc.OrderSide == "buy" {
		bitcloutChange = partialQuantityProcessed - (partialQuantityProcessed * global.Exchange.FEE)
		etherChange = -execPrice / ETHUSD
	} else {
		bitcloutChange = -partialQuantityProcessed
		etherChange = (execPrice - (execPrice * global.Exchange.FEE)) / ETHUSD
	}

	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update := bson.M{"$set": bson.M{"orderQuantityProcessed": partialQuantityProcessed, "execPrice": (execPrice / partialQuantityProcessed)}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}

	return nil
}

/*
Completes a market order

Arguments:
	ctx - The context from which the function is being called
	orderID - The ID of the order to complete
	auantityProcessed - The quantity of the total order that has been processed
	execPrice - The price at which the limit order is executed (sold or bought)

Returns:
	An error if:
		(most likely) the market order could not be partially fulfilled
		mongo connection failed or something went wrong.
*/
func MarketOrder(ctx context.Context, orderID string, quantityProcessed float64, totalPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("market fulfill: %v - %v\n", orderID, quantityProcessed)
	var orderDoc *models.OrderSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println("Couldn't find the orderID\n" + err.Error())
		return err
	}
	var bitcloutChange, etherChange float64

	if orderDoc.OrderSide == "buy" {
		bitcloutChange = quantityProcessed - (quantityProcessed * global.Exchange.FEE)
		etherChange = -totalPrice / ETHUSD
	} else {
		bitcloutChange = -quantityProcessed
		etherChange = (totalPrice - (totalPrice * global.Exchange.FEE)) / ETHUSD
	}

	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update := bson.M{"$set": bson.M{"orderQuantityProcessed": quantityProcessed, "execPrice": (totalPrice / quantityProcessed), "complete": true, "completeTime": time.Now().UTC()}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}
