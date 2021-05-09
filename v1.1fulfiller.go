package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	helmet "github.com/danielkov/gin-helmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	ob "v1.1-fulfiller/orderbook"
)

var exchange = ob.NewOrderBook()

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Exchange Manager")
}

func main() {
	exchange.ProcessLimitOrder(ob.Sell, "uinqueID", decimal.New(55, 0), decimal.New(100, 0))
	exchange.ProcessLimitOrder(ob.Buy, "uinqubvvbeID", decimal.New(100, 0), decimal.New(10, 0))
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	router := gin.Default()

	router.Use(cors.Default())
	router.Use(helmet.Default())
	// router.Use(internalServerAuth())

	f, _ := os.Create("out.log")
	gin.DefaultWriter = io.MultiWriter(f)

	router.GET("/", rootHandler)
	exchangeRouter := router.Group("/exchange")
	// exchangeRouter := router.Group("/exchange",internalServerAuth())
	exchangeRouter.POST("/market", MarketOrderHandler)
	exchangeRouter.POST("/limit", LimitOrderHandler)
	exchangeRouter.POST("/cancel", CancelOrderHandler)
	// exchangeRouter.POST("/market-price", GetMarketPriceHandler)
	router.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotFound)
	})

	fmt.Printf("Starting server at port 5050\n")
	if err := router.Run("localhost:5050"); err != nil {
		log.Fatal(err)
	}
}
