package global

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
)

var ETHUSD float64
var Wg sync.WaitGroup
var MongoClient *mongo.Client
var MongoContext context.Context
var MongoContextCancel context.CancelFunc
