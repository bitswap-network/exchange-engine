package global

import (
	"sync"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

var ETHUSD float64
var Wg sync.WaitGroup
var Api Server

const FEE float64 = 0.02

type Server struct {
	Router *gin.Engine
	Mongo  *mongo.Client
}
