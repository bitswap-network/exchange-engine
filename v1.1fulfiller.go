package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	_ "github.com/shopspring/decimal"
	_ "v1.1-fulfiller/orderbook"
)


func rootHandler(c *gin.Context) {
	c.String(http.StatusOK, "Bitswap Order Manager")
}

func main() {
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
