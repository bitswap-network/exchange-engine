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
	model "v1.1-fulfiller/models"
	ob "v1.1-fulfiller/orderbook"
)

var exchange = ob.NewOrderBook()

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Exchange Manager")
}


func createMarketHandler(c *gin.Context){
	//estimate price?

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
	order.OrderType = "market"
	order.Created = time.Now()
	order.OrderID = fmt.Sprintf("market-%s-%s-%v-%v", order.OrderSide, order.Username, order.OrderQuantity, order.Created.UnixNano()/ int64(time.Millisecond))
	saveErr := CreateOrder(&order)
	if saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg":saveErr})
		return
	}

	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	if orderQuantity.Sign() <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"msg":ob.ErrInvalidQuantity})
		return
	}
	ordersDone, partialDone, partialQuantityProcessed,quantityLeft,totalPrice, error := exchange.ProcessMarketOrder(orderSide, orderQuantity)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": error})
		return
	}	
	if ordersDone != nil {
		ProcessFull(ordersDone)
		if partialDone != nil {
		ProcessPartial(partialDone, partialQuantityProcessed)
		}
	}
	tP,_ := totalPrice.Float64()
	if quantityLeft.IsPositive() {
		err := PartialFulfillOrder(order.OrderID, order.OrderQuantityProcessed,tP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
    	log.Fatal(err)
			return
  	}
	} else{
		err := FulfillOrder(order.OrderID,tP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
    	log.Fatal(err)
			return
  	}	
	}
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
}

func ProcessFull(orderlist []*ob.Order){
	for _,order := range orderlist {
		err := FulfillOrder(order.ID(),0)
		if err != nil {
    	log.Fatal(err)
  	}
	}
}

func ProcessPartial(order *ob.Order, partialQuantityProcessed decimal.Decimal){
		pQ,_ := partialQuantityProcessed.Float64()
		err := PartialFulfillOrder(order.ID(),pQ,0)
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
	exchangeRouter.POST("/exchange/market", createMarketHandler)
	router.NoRoute(func(c *gin.Context) {
        c.AbortWithStatus(http.StatusNotFound)
    })
	
	fmt.Printf("Starting server at port 5050\n")
	if err := router.Run("localhost:5050"); err != nil {
		log.Fatal(err)
	}
}
