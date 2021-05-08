package main

import (
	// "context"
	// "encoding/json"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	// "go.mongodb.org/mongo-driver/bson"
	// "go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	sb "v1.1-fulfiller/orderbook"
)

var clientInstance *mongo.Client
//Used during creation of singleton client object in GetMongoClient().
var clientInstanceError error
//Used to execute client creation procedure only once.
var mongoOnce sync.Once
//I have used below constants just to hold required database config's.
const (
	CONNECTIONSTRING = "mongodb+srv://admin:irxvNRi4aYEmLYBG@bitswap.lernp.mongodb.net"
	DB               = "bitswap"
)

func GetMongoClient() (*mongo.Client, error) {
	//Perform connection creation operation only once.
	mongoOnce.Do(func() {
		// Set client options
		clientOptions := options.Client().ApplyURI(CONNECTIONSTRING)
		// Connect to MongoDB
		client, err := mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			clientInstanceError = err
		}
		// Check the connection
		err = client.Ping(context.TODO(), nil)
		if err != nil {
			clientInstanceError = err
		}
		clientInstance = client
	})
	return clientInstance, clientInstanceError
}


func formHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	fmt.Fprintf(w, "POST request successful")
	name := r.FormValue("name")
	address := r.FormValue("address")
	fmt.Fprintf(w, "Name = %s\n", name)
	fmt.Fprintf(w, "Address = %s\n", address)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/hello" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "Hello!")
}

func main() {
	

	orderBook := sb.NewOrderBook()
	fmt.Println(orderBook)
	orderBook.ProcessLimitOrder(sb.Buy, "orjfworijwe",decimal.New(50,0),decimal.New(440,5))
	fmt.Println(orderBook)
	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileServer)
	http.HandleFunc("/form", formHandler)
	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/pools",getAllPools)

	fmt.Printf("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
