package db

import (
	"context"
	"errors"
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
	userDoc - The struct to hold the user document
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
Updates a user's BitClout and Ether balances by `bitcloutChange` and `etherChange` respectively.

One of `bitcloutChange` and `etherChange` MUST BE NEGATIVE. The other MUST BE POSITIVE.
*/
func UpdateUserBalance(ctx context.Context, username string, bitcloutChange, etherChange float64) error {
	if (bitcloutChange > 0) == (etherChange > 0) {
		return errors.New("Both `bitcloutChange` and `etherChange` cannot be positive or negative")
	}
	update := bson.M{"$inc": bson.M{"balance.bitclout": bitcloutChange, "balance.ether": etherChange}}
	_, err := UserCollection().UpdateOne(ctx, bson.M{"bitclout.username": username}, update)
	if err != nil {
		return err
	}
	return nil
}

func GetUserOrders(ctx context.Context, username string) ([]*models.OrderSchema, error) {
	log.Printf("fetching user orders: %v\n", username)
	var ordersArray []*models.OrderSchema
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
		ordersArray = append(ordersArray, &elem)
	}
	return ordersArray, nil
}

func ValidateOrder(ctx context.Context, username string, orderSide string, orderQuantity float64, totalEth float64) bool {
	log.Printf("fetching user balance from: %v\n", username)
	// var userDoc *models.UserSchema
	userDoc, err := GetUserDoc(ctx, username)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	if userDoc.Balance.InTransaction || orderQuantity > 500 || orderQuantity <= 0 {
		return false
	} else {
		if orderSide == "buy" {
			return totalEth <= userDoc.Balance.Ether
		} else {
			return orderQuantity <= userDoc.Balance.Bitclout
		}
	}
}

func GetUserBalance(ctx context.Context, username string) (balance *models.UserBalance, err error) {
	log.Printf("fetching user balance from: %v\n", username)
	// var userDoc *models.UserSchema
	userDoc, err := GetUserDoc(ctx, username)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	log.Println("done fetching balance")
	return userDoc.Balance, nil
}

func CheckUserTransactionState(ctx context.Context, username string) (bool, error) {
	// var userDoc *models.UserSchema
	userDoc, err := GetUserDoc(ctx, username)
	if err != nil {
		log.Println(err.Error())
		return false, err
	}
	return userDoc.Balance.InTransaction, nil
}

func CreateOrder(ctx context.Context, order *models.OrderSchema) error {
	log.Printf("create order: %v \n", order.OrderID)
	inTransaction, err := CheckUserTransactionState(ctx, order.Username)
	if err != nil {
		return err
	}
	if inTransaction {
		return errors.New("User in transaction.")
	} else {
		order.ID = primitive.NewObjectID()
		_, err := OrderCollection().InsertOne(ctx, order)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		log.Println("done creating order")
	}
	return nil
}

func CancelCompleteOrder(ctx context.Context, orderID string, errorString string) error {
	log.Printf("cancel complete: %v\n", orderID)

	update := bson.M{"$set": bson.M{"error": errorString, "complete": true, "completeTime": time.Now().UTC()}}
	_, err := OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func CompleteLimitOrder(ctx context.Context, orderID string, totalPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD

	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *models.OrderSchema

	//Find order in database
	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}

	var bitcloutChange, etherChange, fees float64
	//update ether USD price var
	var quantityDelta = (orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed)
	if orderDoc.OrderSide == "buy" {
		fees = (quantityDelta * global.Exchange.FEE)
		bitcloutChange = quantityDelta - fees
		etherChange = -(totalPrice / ETHUSD)
	} else {
		fees = (totalPrice * global.Exchange.FEE) / ETHUSD
		bitcloutChange = -quantityDelta
		etherChange = (totalPrice / ETHUSD) - fees
	}

	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	execPrice := totalPrice / orderDoc.OrderQuantity
	// Mark the order as complete after bitclout and eth balances are modified
	// We can set `orderQuantityProcessed` since this order is completed.
	update := bson.M{"$set": bson.M{
		"orderQuantityProcessed": orderDoc.OrderQuantity,
		"complete":               true,
		"completeTime":           time.Now().UTC(),
		"execPrice":              execPrice,
	}, "$inc": bson.M{"fees": fees, "etherQuantity": (totalPrice / ETHUSD)}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}

	return nil
}

func CompleteLimitOrderDirect(ctx context.Context, orderID string) error {
	ETHUSD := global.Exchange.ETHUSD

	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *models.OrderSchema

	//Finding order in database
	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}

	var bitcloutChange, etherChange, fees float64
	//update ether USD price var
	if orderDoc.OrderSide == "buy" {
		fees = ((orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) * global.Exchange.FEE)
		bitcloutChange = (orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) - fees
		etherChange = -(((orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) * orderDoc.OrderPrice) / ETHUSD)
	} else {
		fees = (((orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) * orderDoc.OrderPrice) * global.Exchange.FEE) / ETHUSD
		bitcloutChange = -(orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed)
		etherChange = (((orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) * orderDoc.OrderPrice) / ETHUSD) - fees
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
		"execPrice":              orderDoc.OrderPrice,
	}, "$inc": bson.M{"fees": fees, "etherChange": (((orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) * orderDoc.OrderPrice) / ETHUSD)}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}

	return nil
}

/*
Partially Complete a Limit Order
*/
func PartialLimitOrder(ctx context.Context, orderID string, quantityDelta float64, totalPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("partial fulfill: %v - %v\n", orderID, quantityDelta)
	var orderDoc *models.OrderSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err)
		return err
	}
	var bitcloutChange, etherChange, fees float64
	if orderDoc.OrderSide == "buy" {
		fees = quantityDelta * global.Exchange.FEE
		bitcloutChange = quantityDelta - fees
		etherChange = -totalPrice / ETHUSD
	} else {
		fees = (totalPrice * global.Exchange.FEE) / ETHUSD
		bitcloutChange = -quantityDelta
		etherChange = (totalPrice / ETHUSD) - fees
	}
	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	execPrice := (orderDoc.ExecPrice*orderDoc.OrderQuantityProcessed + totalPrice) / (quantityDelta + orderDoc.OrderQuantityProcessed)
	// Mark the order as complete after bitclout and eth balances are modified
	update := bson.M{"$set": bson.M{"execPrice": execPrice}, "$inc": bson.M{"fees": fees, "orderQuantityProcessed": quantityDelta, "etherQuantity": (totalPrice / ETHUSD)}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func PartialLimitOrderDirect(ctx context.Context, orderID string, quantityDelta float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("partial fulfill: %v - %v\n", orderID, quantityDelta)
	var orderDoc *models.OrderSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err)
		return err
	}

	var bitcloutChange, etherChange, fees float64
	if orderDoc.OrderSide == "buy" {
		fees = quantityDelta * global.Exchange.FEE
		bitcloutChange = quantityDelta - fees
		etherChange = -(quantityDelta * orderDoc.OrderPrice) / ETHUSD
	} else {
		fees = ((quantityDelta * orderDoc.OrderPrice) * global.Exchange.FEE) / ETHUSD
		bitcloutChange = -quantityDelta
		etherChange = ((quantityDelta * orderDoc.OrderPrice) / ETHUSD) - fees
	}

	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	// INCREMENT the `orderQuantityProcessed` to reflect the partial quantity processed
	update := bson.M{"$set": bson.M{"execPrice": orderDoc.OrderPrice}, "$inc": bson.M{"fees": fees, "orderQuantityProcessed": quantityDelta, "etherQuantity": ((quantityDelta * orderDoc.OrderPrice) / ETHUSD)}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func MarketOrder(ctx context.Context, orderID string, quantityProcessed float64, totalPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("Fulfilling market order `%s` - Processed: %v\n", orderID, quantityProcessed)
	var orderDoc *models.OrderSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Printf("Error fetching order `%s`: \n"+err.Error(), orderID)
		return err
	}
	var bitcloutChange, etherChange, fees float64
	if orderDoc.OrderSide == "buy" {
		fees = (quantityProcessed * global.Exchange.FEE)
		bitcloutChange = quantityProcessed - fees
		etherChange = -totalPrice / ETHUSD
	} else {
		fees = (totalPrice * global.Exchange.FEE) / ETHUSD
		bitcloutChange = -quantityProcessed
		etherChange = (totalPrice / ETHUSD) - fees
	}

	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update := bson.M{"$set": bson.M{"etherQuantity": (totalPrice / ETHUSD), "fees": fees, "orderQuantityProcessed": quantityProcessed, "execPrice": (totalPrice / quantityProcessed), "complete": true, "completeTime": time.Now().UTC()}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}
