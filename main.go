package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	helmet "github.com/danielkov/gin-helmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/jasonlvhit/gocron"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	db "v1.1-fulfiller/db"
	global "v1.1-fulfiller/global"
	ob "v1.1-fulfiller/orderbook"
)

var exchange = ob.NewOrderBook()
var ENV_MODE string

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Exchange Manager")
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	ENV_MODE = os.Getenv("ENV_MODE")
	gin.SetMode(ENV_MODE)
	SetETHUSD()
	// Uncomment to use S3 saved orderbook state on launch
	recoverOrderbook := GetOrderbookS3()
	if recoverOrderbook != nil {
		log.Println("unmarshalling fetched orderbook")
		err = exchange.UnmarshalJSON(recoverOrderbook)
		if err != nil {
			log.Println("Error loading fetched orderbook")
		}
		log.Println(exchange.String())
	}
}

func RouterSetup() *gin.Engine {
	router := gin.Default()

	router.Use(cors.Default())
	router.Use(helmet.Default())

	router.GET("/", rootHandler)
	router.GET("/market-price/:side/:quantity", GetMarketPriceHandler)
	router.GET("/ethusd", GetETHUSDHandler)
	
	//Debug mode bypasses server auth
	exchangeRouter := router.Group("/exchange", internalServerAuth())

	exchangeRouter.POST("/market", MarketOrderHandler)
	exchangeRouter.POST("/limit", LimitOrderHandler)
	exchangeRouter.POST("/cancel", CancelOrderHandler)
	exchangeRouter.POST("/sanitize", SanitizeHandler)

	router.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotFound)
	})
	return router
}

func InitOrders(log bool) {
	exchange.ProcessLimitOrder(ob.Sell, "uinqueID", decimal.New(50, 0), decimal.New(115, 0))

	exchange.ProcessLimitOrder(ob.Sell, "uinqueID1", decimal.New(100, 0), decimal.New(110, 0))
	exchange.ProcessLimitOrder(ob.Buy, "uinqubvvbeID", decimal.New(100, 0), decimal.New(90, 0))
	exchange.ProcessLimitOrder(ob.Buy, "uinqubvvbeID1", decimal.New(50, 0), decimal.New(85, 0))
	if log {
		fmt.Println(exchange)
	}

}

func main() {
	go func() {
		// Uncomment to run orderbook S3 backup script
		// gocron.Every(30).Minutes().Do(UploadToS3, getOrderbookBytes(), "orderbook")
		gocron.Every(5).Seconds().Do(SetETHUSD)
		gocron.Every(5).Minutes().Do(LogDepth)
		<-gocron.Start()
	}()
	//Adding test orders to book

	port := os.Getenv("PORT")

	if port == "" {
		port = "5050"
		log.Println("$PORT must be set")
	}
	//Must wait for mongo to connect before doing orderbook ops
	client, cancel := db.MongoConnect()
	defer cancel()
	// global.MongoSession.SetMode(mgo.Monotonic, true)
	global.Api = global.Server{
		Router: RouterSetup(),
		Mongo:  client,
	}
	InitOrders(true)
	fmt.Printf("Starting server at port %s\n", port)
	fmt.Println(os.Getenv("GIN_MODE"))
	if err := global.Api.Router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
