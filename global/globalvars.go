package global

import (
	"net/http"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	Api       Server
	WaitGroup sync.WaitGroup
	ETHUSD    float64
)

const FEE float64 = 0.02

type Server struct {
	Server *http.Server
	Mongo  *mongo.Client
}
