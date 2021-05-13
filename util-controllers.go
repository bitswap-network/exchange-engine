package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	ob "v1.1-fulfiller/orderbook"
)

func GetMarketPriceHandler(c *gin.Context) {
	quantityParam := c.Param("quantity")
	sideParam := c.Param("side")
	quantity, err := decimal.NewFromString(quantityParam)
	if err != nil {
		log.Println(err)
		quantity, _ = decimal.NewFromString("10")
	}
	var orderSide ob.Side
	if sideParam == "buy" {
		orderSide = ob.Buy
	} else if sideParam == "sell" {
		orderSide = ob.Sell
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid side"})
		return
	}

	price, err := exchange.CalculateMarketPrice(orderSide, quantity)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
		return
	}
	c.SecureJSON(http.StatusOK, gin.H{"quantity": quantity.String(), "price": price.String(), "side": sideParam})
}
func GetETHUSDHandler(c *gin.Context) {
	c.SecureJSON(http.StatusOK, gin.H{"result": fmt.Sprintf("%f", ETHUSD)})
}
