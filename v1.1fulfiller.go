package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	ob "v1.1-fulfiller/orderbook"
)

func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Order Manager")
}

func main() {
	orderBook := ob.NewOrderBook()
  fmt.Println(orderBook)
	done, partial, partialQuantityProcessed,error := orderBook.ProcessLimitOrder(ob.Sell, "uinqueID", decimal.New(55, 0), decimal.New(100, 0))
	if error != nil {
    log.Fatal(error)
  }
	fmt.Print(done,partial,partialQuantityProcessed,"\n")
	done, partial, partialQuantityProcessed,quantityLeft, error := orderBook.ProcessMarketOrder(ob.Buy, decimal.New(55, 0))
	if error != nil {
    log.Fatal(error)
  }
	fmt.Print(done,partial,partialQuantityProcessed,quantityLeft,"\n")
	err := godotenv.Load()
	if err != nil {
    log.Fatal("Error loading .env file")
  }
	router := gin.Default()
	f, _ := os.Create("out.log")
    gin.DefaultWriter = io.MultiWriter(f)
	
	router.GET("/", rootHandler)
	// router.GET("/pools",getAllPools)
	
	fmt.Printf("Starting server at port 8080\n")
	if err := router.Run("localhost:5000"); err != nil {
		log.Fatal(err)
	}
}
