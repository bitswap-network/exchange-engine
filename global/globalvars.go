package global

import (
	"sync"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

var ETHUSD float64
var Wg sync.WaitGroup
var Api Server

type Server struct {
	Router *gin.Engine
	Mongo  *mongo.Client
}
