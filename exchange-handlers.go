package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	model "v1.1-fulfiller/models"
	ob "v1.1-fulfiller/orderbook"
)

func MarketOrderHandler(c *gin.Context) {
	var order model.OrderSchema
	if err := c.ShouldBindJSON(&order); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	data, _ := json.MarshalIndent(order, "", "  ")
	fmt.Println(string(data))

	var orderSide ob.Side
	if order.OrderSide == "buy" {
		orderSide = ob.Buy
	} else if order.OrderSide == "sell" {
		orderSide = ob.Sell
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid side"})
		return
	}

	order.OrderType = "market"
	order.Created = time.Now()
	order.Complete = false
	order.OrderID = OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)
	saveErr := CreateOrder(&order)
	if saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": saveErr})
		return
	}

	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	if orderQuantity.Sign() <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": ob.ErrInvalidQuantity})
		return
	}
	ordersDone, partialDone, partialQuantityProcessed, quantityLeft, totalPrice, error := exchange.ProcessMarketOrder(orderSide, orderQuantity)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": error})
		return
	}
	log.Println(exchange)
	log.Println(ordersDone, partialDone, partialQuantityProcessed, quantityLeft, totalPrice, error)

	if ordersDone != nil {
		ProcessFull(ordersDone)
		if partialDone != nil {
			ProcessPartial(partialDone, partialQuantityProcessed)
		}
	}
	tP, _ := totalPrice.Float64()
	if quantityLeft.IsPositive() {
		err := PartialFulfillOrder(order.OrderID, order.OrderQuantityProcessed, tP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
			log.Fatal(err)
			return
		}
	} else {
		err := FulfillOrder(order.OrderID, tP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
			log.Fatal(err)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
}

func LimitOrderHandler(c *gin.Context) {
	var order model.OrderSchema
	if err := c.ShouldBindJSON(&order); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	data, _ := json.MarshalIndent(order, "", "  ")
	fmt.Println(string(data))

	var orderSide ob.Side
	if order.OrderSide == "buy" {
		orderSide = ob.Buy
	} else if order.OrderSide == "sell" {
		orderSide = ob.Sell
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid side"})
		return
	}

	order.OrderType = "limit"
	order.Created = time.Now()
	order.Complete = false
	order.OrderID = OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)
	saveErr := CreateOrder(&order)
	if saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": saveErr})
		return
	}

	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	orderPrice := decimal.NewFromFloat(order.OrderPrice)
	if orderQuantity.Sign() <= 0 || orderPrice.Sign() <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": ob.ErrInvalidQuantity})
		return
	}
	ordersDone, partialDone, partialQuantityProcessed, error := exchange.ProcessLimitOrder(orderSide, order.OrderID, orderQuantity, orderPrice)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": error})
		return
	}
	log.Println(exchange)
	log.Println(ordersDone, partialDone, partialQuantityProcessed, error)

	if ordersDone != nil {
		ProcessFull(ordersDone)
	}
	if partialDone != nil {
		ProcessPartial(partialDone, partialQuantityProcessed)
	}
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
}

func CancelOrderHandler(c *gin.Context) {
	var orderID struct {
		ID string `json:"orderID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&orderID); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}
	cancelledOrderId := exchange.CancelOrder(orderID.ID)
	if cancelledOrderId == nil {
		c.String(http.StatusConflict, "Invalid order ID")
		return
	}
	err := CancelCompleteOrder(orderID.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": cancelledOrderId})
}
