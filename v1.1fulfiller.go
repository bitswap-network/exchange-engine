package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	helmet "github.com/danielkov/gin-helmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"gopkg.in/mgo.v2/bson"
	model "v1.1-fulfiller/models"
	ob "v1.1-fulfiller/orderbook"
)

var exchange = ob.NewOrderBook()

const database, collection = "bitswap", "orders"

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Exchange Manager")
}


func createMarketHandler(c *gin.Context){
	var order model.OrderSchema
	if err := c.ShouldBindJSON(&order); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	var orderSide ob.Side
	if order.OrderSide == "buy"{
		orderSide = ob.Buy
	} else if order.OrderSide == "sell" {
		orderSide = ob.Sell
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid side"})
		return
	}
	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	if orderQuantity.Sign() <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"msg":ob.ErrInvalidQuantity})
		return
	}
	ordersDone, partialDone, partialQuantityProcessed,_, error := exchange.ProcessMarketOrder(orderSide, orderQuantity)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": error})
		return
	}
	order.Created = time.Now()
	order.OrderID = fmt.Sprintf("market-%s-%s-%s-%s", order.OrderSide,order.Username, order.OrderQuantity, order.Created.UnixNano()/ int64(time.Millisecond))
	order.PartialQuantityProcessed,_ = partialQuantityProcessed.Float64()
	order.OrderQuantity,_ = orderQuantity.Sub(partialQuantityProcessed).Float64()
	saveErr := CreateOrder(&order)
	if saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg":saveErr})
		return
	}
	if ordersDone != nil {
		ProcessDone(ordersDone)
		return
	}
	if partialDone != nil {
		ProcessPartial(partialDone, partialQuantityProcessed)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
}

func ProcessDone(orderlist []*ob.Order){
	orders := orderlist
	for _,order := range orders {
		err := UpdateOrder(order.ID(), bson.M{"complete":true})
		if err != nil {
    log.Fatal(err)
  	}
	}
}
func ProcessPartial(order *ob.Order, partialQuantityProcessed decimal.Decimal){
		oQ,_ := order.Quantity().Float64()
		pQ,_ := partialQuantityProcessed.Float64()
		err := UpdateOrder(order.ID(), bson.M{"orderQuantity":oQ,"partialQuantityProcessed":pQ})
		if err != nil {
    log.Fatal(err)
  	}
	
}

func main() {
	
	err := godotenv.Load()
	if err != nil {
    log.Fatal("Error loading .env file")
  }
	
	router := gin.Default()

	router.Use(cors.Default())
	router.Use(helmet.Default())

	f, _ := os.Create("out.log")
    gin.DefaultWriter = io.MultiWriter(f)
	
	router.GET("/", rootHandler)

	exchangeRouter := router.Group("/exchange") 
	exchangeRouter.PUT("/exchange/market", createMarketHandler)
	router.NoRoute(func(c *gin.Context) {
        c.AbortWithStatus(http.StatusNotFound)
    })
	
	fmt.Printf("Starting server at port 5050\n")
	if err := router.Run("localhost:5050"); err != nil {
		log.Fatal(err)
	}
}
