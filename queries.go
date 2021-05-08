package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
)

func getAllPools(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("get pools.\n")
	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	filter := bson.M{} //bson.D{{}} specifies 'all documents'
	// pools := []models.Pool{}
	client, err := GetMongoClient()
	if err != nil {
		log.Fatal(err)
		return
	}
	collection := client.Database(DB).Collection("pools")

	if err != nil {
		log.Fatal(err)
		return
	}
	cur, findError := collection.Find(context.TODO(), filter)

if findError != nil {
	log.Fatal(findError)
		return
	}
	var pools []bson.M
	if err = cur.All(context.TODO(), &pools); err != nil {
    log.Fatal(err)
}
fmt.Println(pools)


	json.NewEncoder(w).Encode(pools) // encode similar to serialize process.
}