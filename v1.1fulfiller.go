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
	ob "v1.1-fulfiller/orderbook"
)

var exchange = ob.NewOrderBook()

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Exchange Manager")
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
	exchangeRouter.POST("/exchange/market", MarketOrderHandler)
	router.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotFound)
	})

	fmt.Printf("Starting server at port 5050\n")
	if err := router.Run("localhost:5050"); err != nil {
		log.Fatal(err)
	}
}
