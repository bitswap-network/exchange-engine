package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/shopspring/decimal"
	"v1.1-fulfiller/db"
	"v1.1-fulfiller/models"
	"v1.1-fulfiller/orderbook"
	"v1.1-fulfiller/s3"
)

func SanitizeHandler(c *gin.Context) {
	var reqBody models.UsernameRequest
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
	var orderList []*orderbook.Order
	for _, order := range orders {
		orderFromState := orderbook.GetOrder(order.OrderID)
		log.Println(orderFromState)
		if orderFromState != nil {
			orderList = append(orderList, orderFromState)
		}
	}
	go orderbook.Sanitize(orderList)
	c.String(http.StatusOK, "OK")
	return
}

func MarketOrderHandler(c *gin.Context) {
	var order models.OrderSchema
	if err := c.ShouldBindBodyWith(&order, binding.JSON); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orderSide orderbook.Side
	if order.OrderSide == "buy" {
		orderSide = orderbook.Buy
	} else if order.OrderSide == "sell" {
		orderSide = orderbook.Sell
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": orderbook.ErrInvalidQuantity.Error()})
		return
	}
	ordersDone, partialDone, partialQuantityProcessed, quantityLeft, totalPrice, error := orderbook.ProcessMarketOrder(orderSide, orderQuantity)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	totalPriceFloat, _ := totalPrice.Float64()
	quantityLeftFloat, _ := quantityLeft.Float64()
	partialQuantityProcessedFloat, _ := partialQuantityProcessed.Float64()
	// error = db.UpdateOrderPrice(c.Request.Context(), order.OrderID, totalPriceFloat/order.OrderQuantity)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	// If any orders have been fulfilled, process them
	if len(ordersDone) > 0 {
		go ProcessFull(ordersDone,order.OrderPrice)
	}
	// If any orders have been partially fulfilled, process them
	if partialDone != nil {
		go ProcessPartial(partialDone, partialQuantityProcessedFloat,order.OrderPrice)
	}

	// if the current order has only been partially fulfilled (quantity left > 0), then partially process it
	if quantityLeft.IsPositive() {
		go db.PartialFulfillOrder(context.TODO(), order.OrderID, order.OrderQuantity-quantityLeftFloat, totalPriceFloat,totalPriceFloat/order.OrderQuantity)
	} else {
		//add checks & validators
		go db.FulfillOrder(context.TODO(), order.OrderID, totalPriceFloat,totalPriceFloat/order.OrderQuantity)
	}
	
	go orderbook.SanitizeUsersOrders(order.Username)
	go s3.UploadToS3(orderbook.GetOrderbookBytes())
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
	return
}

func LimitOrderHandler(c *gin.Context) {
	var order models.OrderSchema
	if err := c.ShouldBindBodyWith(&order, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orderSide orderbook.Side
	if order.OrderSide == "buy" {
		orderSide = orderbook.Buy
	} else if order.OrderSide == "sell" {
		orderSide = orderbook.Sell
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": orderbook.ErrInvalidQuantity.Error()})
		return
	}

	ordersDone, partialDone, partialQuantityProcessed, error := orderbook.ProcessLimitOrder(orderSide, order.OrderID, orderQuantity, orderPrice)
	partialQuantityProcessedFloat, _ := partialQuantityProcessed.Float64()
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	if ordersDone != nil {
		go ProcessFull(ordersDone,order.OrderPrice)
	}
	if partialDone != nil {
		go ProcessPartial(partialDone, partialQuantityProcessedFloat,order.OrderPrice)
	}
	go orderbook.SanitizeUsersOrders(order.Username)
	go s3.UploadToS3(orderbook.GetOrderbookBytes())
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
	cancelledOrderId := orderbook.CancelOrder(orderID.ID)
	if cancelledOrderId == nil {
		c.String(http.StatusConflict, "Invalid order ID")
		return
	}

	go db.CancelCompleteOrder(context.TODO(), orderID.ID, "Order Cancelled by User")
	go orderbook.SanitizeUsersOrders(cancelledOrderId.User())
	go s3.UploadToS3(orderbook.GetOrderbookBytes())
	c.JSON(http.StatusOK, gin.H{"order": cancelledOrderId})
	return
}
