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
var ETHUSD float64
var ENV_MODE string

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Exchange Manager")
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	ENV_MODE = os.Getenv("ENV_MODE")
	gin.SetMode(ENV_MODE)
	SetETHUSD()
	// Uncomment to use S3 saved orderbook state on launch
	// recoverOrderbook := GetOrderbookS3()
	// if recoverOrderbook != nil {
	// 	log.Println("unmarshalling fetched orderbook")
	// 	err = exchange.UnmarshalJSON(recoverOrderbook)
	// 	if err != nil {
	// 		log.Println("Error loading fetched orderbook")
	// 	}
	// }
}

func main() {

	go func() {
		// Uncomment to run orderbook S3 backup script
		// gocron.Every(60).Seconds().Do(UploadToS3, getOrderbookBytes(), "orderbook")
		gocron.Every(5).Seconds().Do(SetETHUSD)
		<-gocron.Start()
	}()

	//Adding test orders to book
	exchange.ProcessLimitOrder(ob.Sell, "uinqueID", decimal.New(50, 0), decimal.New(115, 0))
	fmt.Println(exchange)
	exchange.ProcessLimitOrder(ob.Sell, "uinqueID1", decimal.New(100, 0), decimal.New(110, 0))
	fmt.Println(exchange)
	exchange.ProcessLimitOrder(ob.Buy, "uinqubvvbeID", decimal.New(100, 0), decimal.New(90, 0))
	fmt.Println(exchange)
	exchange.ProcessLimitOrder(ob.Buy, "uinqubvvbeID1", decimal.New(50, 0), decimal.New(85, 0))
	fmt.Println(exchange)

	router := gin.Default()

	router.Use(cors.Default())
	router.Use(helmet.Default())

	f, _ := os.Create("out.log")
	gin.DefaultWriter = io.MultiWriter(f)

	router.GET("/", rootHandler)
	router.GET("/market-price/:side/:quantity", GetMarketPriceHandler)
	router.GET("/ethusd", GetMarketPriceHandler)

	//Debug mode bypasses server auth
	exchangeRouter := router.Group("/exchange", internalServerAuth())

	exchangeRouter.POST("/market", MarketOrderHandler)
	exchangeRouter.POST("/limit", LimitOrderHandler)
	exchangeRouter.POST("/cancel", CancelOrderHandler)

	router.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotFound)
	})

	fmt.Printf("Starting server at port 5050\n")
	fmt.Println(os.Getenv("GIN_MODE"))
	if err := router.Run("localhost:5050"); err != nil {
		log.Fatal(err)
	}
}
