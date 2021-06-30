package main

import (
	"encoding/json"
	"log"
	"net/http"

	"exchange-engine/fireeye"
	"exchange-engine/global"
	"exchange-engine/orderbook"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func FireEyeStatusHandler(c *gin.Context) {
	c.SecureJSON(http.StatusOK, gin.H{"Code": fireeye.FireEye.Code, "Message": fireeye.FireEye.Message})
	return
}

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

func GetMarketQuantityHandler(c *gin.Context) {
	maxPriceParam := c.Param("maxPrice")
	sideParam := c.Param("side")
	maxPrice, err := decimal.NewFromString(maxPriceParam)
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
	log.Println(orderSide, maxPrice)

	quantity, err := orderbook.CalculateMarketQuantity(orderSide, maxPrice)
	if err != nil {
		log.Println(err)
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	quantityFloat, _ := quantity.Float64()
	c.SecureJSON(http.StatusOK, gin.H{"quantity": quantityFloat, "side": sideParam})
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
