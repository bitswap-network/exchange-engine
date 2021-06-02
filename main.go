package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	helmet "github.com/danielkov/gin-helmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/jasonlvhit/gocron"
	"github.com/joho/godotenv"
	config "v1.1-fulfiller/config"
	"v1.1-fulfiller/db"
	"v1.1-fulfiller/global"
	ob "v1.1-fulfiller/orderbook"
	"v1.1-fulfiller/s3"
)

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Exchange Manager")
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println(err.Error())
	}
	config.Setup()
	global.Setup()
	s3.Setup()
	db.Setup()
	ob.Setup(false)

}

func RouterSetup() *gin.Engine {
	router := gin.Default()
	router.Use(cors.Default())
	router.Use(helmet.Default())
	router.GET("/", rootHandler)
	router.GET("/market-price/:side/:quantity", GetMarketPriceHandler)
	router.GET("/ethusd", GetETHUSDHandler)
	router.GET("/orderbook-state", GetCurrentDepthHandler)

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

func main() {
	gin.SetMode(config.ServerConfig.RunMode)
	go func() {
		gocron.Every(10).Seconds().Do(global.SetETHUSD)
		gocron.Every(5).Minutes().Do(LogDepth)
		gocron.Every(10).Seconds().Do(LogOrderbook)
		gocron.Every(1).Minute().Do(s3.UploadToS3, ob.GetOrderbookBytes())
		<-gocron.Start()
	}()

	routersInit := RouterSetup()
	readTimeout := config.ServerConfig.ReadTimeout
	writeTimeout := config.ServerConfig.WriteTimeout
	addr := config.ServerConfig.Addr
	maxHeaderBytes := 1 << 20

	srv := &http.Server{
		Addr:           addr,
		Handler:        routersInit,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: maxHeaderBytes,
	}
	fmt.Printf("Starting %s server at: %s\n", config.ServerConfig.RunMode, config.ServerConfig.Addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Printf("Listen Err: %s\n", err.Error())
		}
	}()
	quit := make(chan os.Signal)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctxterm, cancelterm := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancelterm()
		close(global.Exchange.ETHUSD)
	}()

	if err := srv.Shutdown(ctxterm); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")
}
