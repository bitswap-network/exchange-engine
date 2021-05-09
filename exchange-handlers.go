package main

import (
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
	order.OrderID = OrderIDGen("market", order.OrderSide, order.Username, order.OrderQuantity, order.Created)
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
