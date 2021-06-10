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

func GetUserOrders(ctx context.Context, username string) ([]models.OrderSchema, error) {
	log.Printf("fetching user orders: %v\n", username)
	var ordersArray []models.OrderSchema
	cursor, err := OrderCollection().Find(ctx, bson.M{"username": username, "complete": false})
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer cursor.Close(ctx)
	// if err = cursor.All(ctx, ordersDoc); err != nil {
	// 	log.Println(err.Error())
	// }
	for cursor.Next(ctx) {
		//Create a value into which the single document can be decoded
		var elem models.OrderSchema
		err := cursor.Decode(&elem)
		if err != nil {
			log.Println(err)
		}
		log.Println(elem)
		ordersArray = append(ordersArray, elem)
	}
	log.Println("done fetching orders")
	return ordersArray, nil
}

func GetUserBalance(ctx context.Context, username string) (balance *models.UserBalance, err error) {
	log.Printf("fetching user balance from: %v\n", username)
	var userDoc *models.UserSchema
	err = UserCollection().FindOne(ctx, bson.M{"bitclout.username": username}).Decode(&userDoc)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	log.Println("done fetching balance")
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

func CreateOrder(ctx context.Context, order *models.OrderSchema) error {
	log.Printf("create order: %v \n", order.OrderID)
	order.ID = primitive.NewObjectID()
	_, err := OrderCollection().InsertOne(ctx, order)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	log.Println("done creating order")
	return nil
}

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

func UpdateOrderPrice(ctx context.Context, orderID string, orderPrice float64) error {
	log.Printf("update order price: %v\n", orderID)

	update := bson.M{"$set": bson.M{"orderPrice": orderPrice}}
	_, err := OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)

	if err != nil {
		return err
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

func CompleteLimitOrder(ctx context.Context, orderID string, execPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD

	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *models.OrderSchema
	var userDoc *models.UserSchema

	//Finding order in database
	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}
	//finding user associated with order
	err = UserCollection().FindOne(ctx, bson.M{"bitclout.username": orderDoc.Username}).Decode(&userDoc)
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
	update := bson.M{"$inc": bson.M{"balance.bitclout": bitcloutChange, "balance.ether": etherChange}}
	_, err = UserCollection().UpdateOne(ctx, bson.M{"bitclout.username": orderDoc.Username}, update)
	if err != nil {
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update = bson.M{"$set": bson.M{
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

func PartialLimitOrder(ctx context.Context, orderID string, partialQuantityProcessed float64, execPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("partial fulfill: %v - %v\n", orderID, partialQuantityProcessed)
	var orderDoc *models.OrderSchema
	var userDoc *models.UserSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err)
		return err
	}
	err = UserCollection().FindOne(ctx, bson.M{"bitclout.username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		log.Println(err)
		return err
	}
	var bitcloutChange, etherChange float64
	if orderDoc.OrderSide == "buy" {
		bitcloutChange = partialQuantityProcessed - (partialQuantityProcessed * global.Exchange.FEE)
		// bitcloutBalanceUpdated = userDoc.Balance.Bitclout + bitcloutChange
		etherChange = -execPrice / ETHUSD
		// etherBalanceUpdated = userDoc.Balance.Ether + etherChange
	} else {
		bitcloutChange = -partialQuantityProcessed
		// bitcloutBalanceUpdated = userDoc.Balance.Bitclout + bitcloutChange
		etherChange = (execPrice - (execPrice * global.Exchange.FEE)) / ETHUSD
		// etherBalanceUpdated = userDoc.Balance.Ether + etherChange
	}

	// attempt to modify bitclout balance and eth balance
	update := bson.M{"$inc": bson.M{"balance.bitclout": bitcloutChange, "balance.ether": etherChange}}
	_, err = UserCollection().UpdateOne(ctx, bson.M{"bitclout.username": orderDoc.Username}, update)
	if err != nil {
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update = bson.M{"$set": bson.M{"orderQuantityProcessed": partialQuantityProcessed, "execPrice": (execPrice / partialQuantityProcessed)}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}

	return nil
}

func MarketOrder(ctx context.Context, orderID string, quantityProcessed float64, totalPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("market fulfill: %v - %v\n", orderID, quantityProcessed)
	var orderDoc *models.OrderSchema
	var userDoc *models.UserSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println("Couldn't find the orderID\n" + err.Error())
		return err
	}
	err = UserCollection().FindOne(ctx, bson.M{"bitclout.username": orderDoc.Username}).Decode(&userDoc)
	if err != nil {
		log.Println("Couldn't find the user\n" + err.Error())
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

	update := bson.M{"$inc": bson.M{"balance.bitclout": bitcloutChange, "balance.ether": etherChange}}
	_, err = UserCollection().UpdateOne(ctx, bson.M{"bitclout.username": orderDoc.Username}, update)
	if err != nil {
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update = bson.M{"$set": bson.M{"orderQuantityProcessed": quantityProcessed, "execPrice": (totalPrice / quantityProcessed), "complete": true, "completeTime": time.Now().UTC()}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}
