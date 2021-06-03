package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/shopspring/decimal"
	db "v1.1-fulfiller/db"
	model "v1.1-fulfiller/models"
	ob "v1.1-fulfiller/orderbook"
)

func SanitizeHandler(c *gin.Context) {
	var reqBody model.UsernameRequest
	if err := c.ShouldBindBodyWith(&reqBody, binding.JSON); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if reqBody.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Username"})
		return
	}
	orders, err := db.GetUserOrders(c.Request.Context(), reqBody.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var orderList []*ob.Order
	for _, order := range orders {
		orderFromState := ob.GetOrder(order.OrderID)
		orderList = append(orderList, orderFromState)
	}
	go ob.Sanitize(orderList)
	c.String(http.StatusOK, "OK")
	return
}

func MarketOrderHandler(c *gin.Context) {
	var order model.OrderSchema
	if err := c.ShouldBindBodyWith(&order, binding.JSON); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orderSide ob.Side
	if order.OrderSide == "buy" {
		orderSide = ob.Buy
	} else if order.OrderSide == "sell" {
		orderSide = ob.Sell
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid side"})
		return
	}

	order.OrderType = "market"
	order.Created = time.Now()
	order.Complete = true
	order.CompleteTime = time.Now()
	order.OrderID = OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)
	db.CreateOrder(c.Request.Context(), &order)
	// add error handling

	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	if orderQuantity.Sign() <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ob.ErrInvalidQuantity.Error()})
		return
	}
	ordersDone, partialDone, partialQuantityProcessed, quantityLeft, totalPrice, error := ob.ProcessMarketOrder(orderSide, orderQuantity)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	totalPriceFloat, _ := totalPrice.Float64()
	quantityLeftFloat, _ := quantityLeft.Float64()
	partialQuantityProcessedFloat, _ := partialQuantityProcessed.Float64()
	error = db.UpdateOrderPrice(c.Request.Context(), order.OrderID, totalPriceFloat/order.OrderQuantity)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	// If any orders have been fulfilled, process them
	if len(ordersDone) > 0 {
		go ProcessFull(ordersDone)
	}
	// If any orders have been partially fulfilled, process them
	if partialDone != nil {
		go ProcessPartial(partialDone, partialQuantityProcessedFloat)
	}

	// if the current order has only been partially fulfilled (quantity left > 0), then partially process it
	if quantityLeft.IsPositive() {
		go db.PartialFulfillOrder(context.TODO(), order.OrderID, order.OrderQuantity-quantityLeftFloat, totalPriceFloat)
	} else {
		//add checks & validators
		go db.FulfillOrder(context.TODO(), order.OrderID, totalPriceFloat)
	}
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
	return
}

func LimitOrderHandler(c *gin.Context) {
	var order model.OrderSchema
	if err := c.ShouldBindBodyWith(&order, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orderSide ob.Side
	if order.OrderSide == "buy" {
		orderSide = ob.Buy
	} else if order.OrderSide == "sell" {
		orderSide = ob.Sell
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid side"})
		return
	}

	order.OrderType = "limit"
	order.Created = time.Now()
	order.Complete = false
	order.OrderID = OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)
	//add error handling
	error := db.CreateOrder(c.Request.Context(), &order)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}

	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	orderPrice := decimal.NewFromFloat(order.OrderPrice)
	if orderQuantity.Sign() <= 0 || orderPrice.Sign() <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ob.ErrInvalidQuantity.Error()})
		return
	}

	ordersDone, partialDone, partialQuantityProcessed, error := ob.ProcessLimitOrder(orderSide, order.OrderID, orderQuantity, orderPrice)
	partialQuantityProcessedFloat, _ := partialQuantityProcessed.Float64()
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	if ordersDone != nil {
		go ProcessFull(ordersDone)
	}
	if partialDone != nil {
		go ProcessPartial(partialDone, partialQuantityProcessedFloat)
	}
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
	return
}

func CancelOrderHandler(c *gin.Context) {
	var orderID struct {
		ID string `json:"orderID" binding:"required"`
	}
	if err := c.ShouldBindBodyWith(&orderID, binding.JSON); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cancelledOrderId := ob.CancelOrder(orderID.ID)
	if cancelledOrderId == nil {
		c.String(http.StatusConflict, "Invalid order ID")
		return
	}
	go db.CancelCompleteOrder(context.TODO(), orderID.ID, "Order Cancelled by User")

	c.JSON(http.StatusOK, gin.H{"order": cancelledOrderId})
	return
}
