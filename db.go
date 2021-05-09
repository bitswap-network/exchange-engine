package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	// "go.mongodb.org/mongo-driver/bson"
	// "go.mongodb.org/mongo-driver/bson/primitive"
	model "v1.1-fulfiller/models"
)
const (
	// Timeout operations after N seconds
	connectTimeout           = 5
	connectionStringTemplate = "mongodb+srv://%s:%s@%s"
	database = "bitswap"

)

// GetConnection - Retrieves a client to the DocumentDB
func mongoConnect() *mgo.Session {
	username := os.Getenv("MONGODB_USERNAME")
	password := os.Getenv("MONGODB_PASSWORD")
	clusterEndpoint := os.Getenv("MONGODB_ENDPOINT")

	connectionURI := fmt.Sprintf(connectionStringTemplate, username, password, clusterEndpoint)

	session, err := mgo.Dial(connectionURI)
	if err != nil {
fmt.Println("session err:", err)
os.Exit(1)
}

	return session
}

func CreateOrder(order *model.OrderSchema) error {
	session := mongoConnect()
	defer session.Close()
	order.ID = bson.NewObjectId()
	err := session.DB(database).C("orders").Insert(order)
	if err != nil {
		log.Printf("Could not create order: %v", err)
		return err
	}
	return nil
}
func GetOrderByOrderId(orderID string) (orderDoc *model.OrderSchema, err error) {
	session := mongoConnect()
	defer session.Close()
	query := bson.M{"orderID":orderID}
	db := session.DB(database)
  collection := db.C("orders")
	qerr := collection.Find(query).One(&orderDoc)
	if err != nil {
		log.Printf("Could not create Task: %v", err)
		return nil, qerr
	}
	return orderDoc, nil
}
func RemoveOrder(selector bson.M) error {
    session := mongoConnect()
    defer session.Close()
    db := session.DB(database)
    collection := db.C("orders")
    err := collection.Remove(selector)
    if err != nil {
        return err
    }
    return nil
}

func FulfillOrder(orderID string) (err error) {
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
    session := mongoConnect()
    defer session.Close()
    db := session.DB(database)
    orders := db.C("orders")
		users := db.C("users")
		
		err = orders.Find(bson.M{"orderID":orderID}).One(&orderDoc)
    if err != nil {
        return err
    }
    err = orders.Update(bson.M{"orderID":orderID}, bson.M{"orderQuantityProcessed":orderDoc.OrderQuantity, "complete":true,"completeTime":time.Now()})
    if err != nil {
        return err
    }
		err = users.Find(bson.M{"username":orderDoc.Username}).One(&userDoc)
    if err != nil {
       return err
    }
		var bitcloutBalanceUpdated float64
		var etherBalanceUpdated float64
		//update ether USD price var
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout+(orderDoc.OrderPrice*orderDoc.OrderQuantity)
			etherBalanceUpdated = userDoc.Balance.Ether-(orderDoc.OrderPrice*orderDoc.OrderQuantity/3000)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout-(orderDoc.OrderPrice*orderDoc.OrderQuantity)
			etherBalanceUpdated = userDoc.Balance.Ether+(orderDoc.OrderPrice*orderDoc.OrderQuantity/3000)
		}

		err = users.Update(bson.M{"username":orderDoc.Username},bson.M{"balance.bitclout":bitcloutBalanceUpdated,"balance.ether":etherBalanceUpdated} )
    if err != nil {
        return err
    }
		// add check for negative balance here
    return nil
}
func PartialFulfillOrder(orderID string, partialQuantityProcessed float64 ) (err error) {
	var orderDoc *model.OrderSchema
	var userDoc *model.UserSchema
    session := mongoConnect()
    defer session.Close()
    db := session.DB(database)
    orders := db.C("orders")
		users := db.C("users")
		// oQP,_ := order.Quantity().Float64()
    err = orders.Update(bson.M{"orderID":orderID}, bson.M{"orderQuantityProcessed":partialQuantityProcessed})
    if err != nil {
        return err
    }
		err = orders.Find(bson.M{"orderID":orderID}).One(&orderDoc)
    if err != nil {
        return err
    }
		err = users.Find(bson.M{"username":orderDoc.Username}).One(&userDoc)
    if err != nil {
       return err
    }
		var bitcloutBalanceUpdated float64
		var etherBalanceUpdated float64
		//update ether USD price var
		if orderDoc.OrderSide == "buy" {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout+(orderDoc.OrderPrice*partialQuantityProcessed)
			etherBalanceUpdated = userDoc.Balance.Ether-(orderDoc.OrderPrice*partialQuantityProcessed/3000)
		} else {
			bitcloutBalanceUpdated = userDoc.Balance.Bitclout-(orderDoc.OrderPrice*partialQuantityProcessed)
			etherBalanceUpdated = userDoc.Balance.Ether+(orderDoc.OrderPrice*partialQuantityProcessed/3000)
		}

		err = users.Update(bson.M{"username":orderDoc.Username},bson.M{"balance.bitclout":bitcloutBalanceUpdated,"balance.ether":etherBalanceUpdated} )
    if err != nil {
        return err
    }
		// check for negative balance here
    return nil
}
