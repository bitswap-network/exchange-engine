package main

import (
	"fmt"
	"log"
	"os"

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
	err := session.DB(database).C(collection).Insert(order)
	if err != nil {
		log.Printf("Could not create order: %v", err)
		return err
	}
	return nil
}
func GetOrderByOrderId(orderID string) (order *model.OrderSchema, err error) {
	session := mongoConnect()
	defer session.Close()
	query := bson.M{"orderID":orderID}
	db := session.DB(database)
    collection := db.C(collection)
	qerr := collection.Find(query).One(&order)
	if err != nil {
		log.Printf("Could not create Task: %v", err)
		return nil, qerr
	}
	return order, nil
}
func RemoveOrder(selector bson.M) error {
    session := mongoConnect()
    defer session.Close()
    db := session.DB(database)
    collection := db.C(collection)
    err := collection.Remove(selector)
    if err != nil {
        return err
    }
    return nil
}

func UpdateOrder(orderID string, update interface{}) error {
    session := mongoConnect()
    defer session.Close()
		query := bson.M{"orderID":orderID}
    db := session.DB(database)
    collection := db.C(collection)
    err := collection.Update(query, update)
    if err != nil {
        return err
    }
    return nil
}