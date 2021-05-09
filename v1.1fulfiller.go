package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	helmet "github.com/danielkov/gin-helmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jasonlvhit/gocron"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	ob "v1.1-fulfiller/orderbook"
)

var exchange = ob.NewOrderBook()
var wg sync.WaitGroup

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Exchange Manager")
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	recoverOrderbook := GetOrderbookS3()
	if recoverOrderbook != nil {
		log.Println("unmarshalling fetched orderbook")
		err = exchange.UnmarshalJSON(recoverOrderbook)
		if err != nil {
			log.Println("Error loading fetched orderbook")
		}
	}
}

func main() {
	// uploadSyncS3 := gocron.NewScheduler()
	gocron.Every(30).Seconds().Do(UploadToS3, getOrderbookBytes(), "orderbook")
	exchange.ProcessLimitOrder(ob.Sell, "uinqueID", decimal.New(55, 0), decimal.New(100, 0))
	exchange.ProcessLimitOrder(ob.Buy, "uinqubvvbeID", decimal.New(100, 0), decimal.New(10, 0))

	router := gin.Default()

	router.Use(cors.Default())
	router.Use(helmet.Default())

	f, _ := os.Create("out.log")
	gin.DefaultWriter = io.MultiWriter(f)

	router.GET("/", rootHandler)
	router.GET("/market-price/:side/:quantity", GetMarketPriceHandler)

	// exchangeRouter := router.Group("/exchange")
	exchangeRouter := router.Group("/exchange", internalServerAuth())
	exchangeRouter.POST("/market", MarketOrderHandler)
	exchangeRouter.POST("/limit", LimitOrderHandler)
	exchangeRouter.POST("/cancel", CancelOrderHandler)
	router.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotFound)
	})
	<-gocron.Start()
	fmt.Printf("Starting server at port 5050\n")
	if err := router.Run("localhost:5050"); err != nil {
		log.Fatal(err)
	}

}
