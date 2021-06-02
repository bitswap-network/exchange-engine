package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	global "v1.1-fulfiller/global"
	ob "v1.1-fulfiller/orderbook"
)

func GetMarketPriceHandler(c *gin.Context) {
	quantityParam := c.Param("quantity")
	sideParam := c.Param("side")
	quantity, err := decimal.NewFromString(quantityParam)
	if err != nil {
		log.Println(err)
		quantity, _ = decimal.NewFromString("1")
	}
	var orderSide ob.Side
	if sideParam == "buy" {
		orderSide = ob.Buy
	} else if sideParam == "sell" {
		orderSide = ob.Sell
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid side"})
		return
	}

	price, err := ob.CalculateMarketPrice(orderSide, quantity)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	quantityFloat, _ := quantity.Float64()
	priceFloat, _ := price.Float64()
	c.SecureJSON(http.StatusOK, gin.H{"quantity": quantityFloat, "price": priceFloat, "side": sideParam})
}

func GetCurrentDepthHandler(c *gin.Context) {
	depthMarshal, err := ob.DepthMarshalJSON()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	jsonMarshall, err := json.Marshal(depthMarshal)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, jsonMarshall)
	return
}

func GetETHUSDHandler(c *gin.Context) {
	ETHUSD := <-global.Exchange.ETHUSD
	// log.Println(global.ETHUSD)
	c.SecureJSON(http.StatusOK, gin.H{"result": ETHUSD})
	return
}
