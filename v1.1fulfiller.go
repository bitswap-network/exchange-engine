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
	orderQuantity := decimal.NewFromFloat32(order.OrderQuantity)
	if orderQuantity.Sign() <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"msg":ob.ErrInvalidQuantity})
		return
	}
	order.Created = time.Now()
	order.OrderID = fmt.Sprintf("%s-%s-%s-%s", order.Username, order.OrderSide, order.OrderQuantity, order.Created.UnixNano()/ int64(time.Millisecond))
	
	
	
	done, partial, partialQuantityProcessed,quantityLeft, error := exchange.ProcessMarketOrder(orderSide, orderQuantity)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": error})
		return
	}
	order.PartialQuantityProcessed = order.PartialQuantityProcessed
	id, saveErr := SaveOrder(&order)
	if saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg":saveErr})
		return
	}
	if done != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": error})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func ProcessDone(orderlist []*ob.Order){
	orders := orderlist
	for i,order := range orders {

	}
}

func main() {
	// exchange = ob.NewOrderBook()
  fmt.Println(exchange)
	done, partial, partialQuantityProcessed,error := exchange.ProcessLimitOrder(ob.Sell, "uinqueID", decimal.New(55, 0), decimal.New(100, 0))
	if error != nil {
    log.Fatal(error)
  }
	fmt.Print(done,partial,partialQuantityProcessed,"\n")
	done, partial, partialQuantityProcessed,quantityLeft, error := exchange.ProcessMarketOrder(ob.Buy, decimal.New(55, 0))
	if error != nil {
    log.Fatal(error)
  }
	fmt.Print(done,partial,partialQuantityProcessed,quantityLeft,"\n")
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
	
	
	// router.GET("/pools",getAllPools)
	
	fmt.Printf("Starting server at port 8080\n")
	if err := router.Run("localhost:5000"); err != nil {
		log.Fatal(err)
	}
}
