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
	if err := c.ShouldBindWith(&order, binding.JSON); err != nil {
		log.Print(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure that the orderSide is "buy" or "sell"
	var orderSide orderbook.Side
	if order.OrderSide == "buy" {
		orderSide = orderbook.Buy
	} else if order.OrderSide == "sell" {
		orderSide = orderbook.Sell
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid side"})
		return
	}
	// Ensure that the order has a valid quantity
	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	if orderQuantity.Sign() <= 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": orderbook.ErrInvalidQuantity.Error()})
		return
	}

	// Initialize the Order
	order.OrderType = "market"
	order.Created = time.Now().UTC()
	order.OrderID = OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)

	// Attempt to create an order in the database
	err := db.CreateOrder(c.Request.Context(), &order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Attempt to Process the Market Order
	quantityLeft, totalPrice, error := orderbook.ProcessMarketOrder(orderSide, orderQuantity)
	log.Println(quantityLeft, totalPrice)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}

	// give the current order's issuer orderQuantity - quantityLeft
	totalPriceFloat, _ := totalPrice.Float64()
	quantityLeftFloat, _ := quantityLeft.Float64()
	error = db.MarketOrder(context.TODO(), order.OrderID, order.OrderQuantity-quantityLeftFloat, totalPriceFloat)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	// go orderbook.SanitizeUsersOrders(order.Username)
	go s3.UploadToS3(orderbook.GetOrderbookBytes())
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
	return
}

func LimitOrderHandler(c *gin.Context) {
	var order models.OrderSchema
	if err := c.ShouldBindWith(&order, binding.JSON); err != nil {
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
	order.Created = time.Now().UTC()
	order.Complete = false
	order.OrderQuantityProcessed = 0
	order.OrderID = OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)

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

	// Attempt to process Limit Order
	quantityLeft, totalPrice, error := orderbook.ProcessLimitOrder(orderSide, order.OrderID, orderQuantity, orderPrice)
	totalPriceFloat, _ := totalPrice.Float64()
	quantityLeftFloat, _ := quantityLeft.Float64()
	log.Println(quantityLeft, totalPrice)
	if error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}

	if quantityLeft.IsPositive() {
		if quantityLeft != orderQuantity {
			// partially fulfilled
			error = db.PartialLimitOrder(c.Request.Context(), order.OrderID, order.OrderQuantity-quantityLeftFloat, totalPriceFloat)
			if error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
				return
			}
		}
	} else {
		error = db.CompleteLimitOrder(c.Request.Context(), order.OrderID, totalPriceFloat)
		if error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
			return
		}
	}
	go s3.UploadToS3(orderbook.GetOrderbookBytes())
	c.JSON(http.StatusOK, gin.H{"id": order.OrderID})
	return
}

func CancelOrderHandler(c *gin.Context) {
	var orderID struct {
		ID string `json:"orderID" binding:"required"`
	}
	if err := c.ShouldBindWith(&orderID, binding.JSON); err != nil {
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
