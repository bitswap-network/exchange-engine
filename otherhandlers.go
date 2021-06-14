package main

import (
	"encoding/json"
	"log"
	"net/http"

	"exchange-engine/global"
	"exchange-engine/orderbook"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func GetMarketPriceHandler(c *gin.Context) {
	quantityParam := c.Param("quantity")
	sideParam := c.Param("side")
	quantity, err := decimal.NewFromString(quantityParam)
	if err != nil {
		log.Println(err)
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var orderSide orderbook.Side
	if sideParam == "buy" {
		orderSide = orderbook.Buy
	} else if sideParam == "sell" {
		orderSide = orderbook.Sell
	} else {
		c.SecureJSON(http.StatusBadRequest, gin.H{})
		return
	}

	price, err := orderbook.CalculateMarketPrice(orderSide, quantity)
	if err != nil {
		log.Println(err)
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	quantityFloat, _ := quantity.Float64()
	priceFloat, _ := price.Float64()
	c.SecureJSON(http.StatusOK, gin.H{"quantity": quantityFloat, "price": priceFloat, "side": sideParam})
	return
}

func GetCurrentDepthHandler(c *gin.Context) {
	depthMarshal, err := orderbook.DepthMarshalJSON()
	if err != nil {
		log.Println(err)
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	jsonMarshall, err := json.Marshal(depthMarshal)
	if err != nil {
		log.Println(err)
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, gin.MIMEJSON, jsonMarshall)
	return
}

func GetETHUSDHandler(c *gin.Context) {
	// log.Println(global.ETHUSD)
	c.SecureJSON(http.StatusOK, gin.H{"result": global.Exchange.ETHUSD})
	return
}
