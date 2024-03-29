package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"exchange-engine/db"
	"exchange-engine/global"
	"exchange-engine/models"
	"exchange-engine/orderbook"
	"exchange-engine/s3"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/shopspring/decimal"
)

func OrderIDGen(orderType string, orderSide string, publicKey string, quantity float64, created time.Time) (orderID string) {
	return fmt.Sprintf("%s-%s-%s-%v-%v", orderType, orderSide, publicKey, quantity, created.UnixNano()/int64(time.Millisecond))
}

func SanitizeHandler(c *gin.Context) {
	var reqBody models.SanitizeRequest
	if err := c.ShouldBindWith(&reqBody, binding.JSON); err != nil {
		log.Print(err)
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if reqBody.PublicKey == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": "Invalid Public Key"})
		return
	}
	orderbook.SanitizeUsersOrders(reqBody.PublicKey)
	c.String(http.StatusOK, "OK")
	return
}

func MarketOrderHandler(c *gin.Context) {
	slippageParam := c.Param("slippage")
	quoteParam := c.Param("quote")
	quote, err := decimal.NewFromString(quoteParam)
	if err != nil {
		log.Println(err)
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	slippage, err := decimal.NewFromString(slippageParam)
	if err != nil {
		log.Println(err)
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var order models.OrderSchema
	var orderSide orderbook.Side

	if err := c.ShouldBindWith(&order, binding.JSON); err != nil {
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure that the orderSide is "buy" or "sell"
	if order.OrderSide == "buy" {
		orderSide = orderbook.Buy
	} else if order.OrderSide == "sell" {
		orderSide = orderbook.Sell
	} else {
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": "invalid side"})
		return
	}

	// Ensure that the order has a valid quantity
	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	if orderQuantity.Sign() <= 0 {
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": orderbook.ErrInvalidQuantity.Error()})
		return
	}

	// Initialize the Order
	order.OrderType = "market"
	order.Created = time.Now().UTC()
	order.OrderID = OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)
	order.OrderQuantityProcessed = 0
	order.EtherQuantity = 0
	order.Fees = 0
	estMarketPrice, err := orderbook.CalculateMarketPrice(orderSide, orderQuantity)
	if err != nil {
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	estMarketPriceFloat, _ := estMarketPrice.Float64()
	if !db.ValidateOrder(c.Request.Context(), order.Username, order.OrderSide, order.OrderQuantity, estMarketPriceFloat/global.Exchange.ETHUSD) {
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": "Could not validate order."})
		return
	}
	orderSlippage := quote.Sub(estMarketPrice).Abs().Div(quote)
	log.Println(quote, slippage, orderSlippage, estMarketPrice)
	if orderSlippage.GreaterThan(slippage) {
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": "Could not execute order without slippage."})
		return
	}
	// if math.Abs(estMarketPriceFloat-order.Quote)
	// Attempt to create an order in the database
	err = db.CreateOrder(c.Request.Context(), &order)
	if err != nil {
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Attempt to Process the Market Order
	quantityLeft, tradePrice, err := orderbook.ProcessMarketOrder(orderSide, orderQuantity)
	log.Println(quantityLeft, tradePrice, err)
	if err != nil {
		db.CancelCompleteOrder(c.Request.Context(), order.OrderID, err.Error())
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// give the current order's issuer `orderQuantity - quantityLeft` (equivalent value as `tradePrice`)
	tradePriceFloat, _ := tradePrice.Float64()
	quantityLeftFloat, _ := quantityLeft.Float64()
	err = db.MarketOrder(context.TODO(), order.OrderID, order.OrderQuantity-quantityLeftFloat, tradePriceFloat)
	if err != nil {
		db.CancelCompleteOrder(c.Request.Context(), order.OrderID, err.Error())
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	go orderbook.SanitizeUsersOrders(order.Username)
	go s3.UploadToS3(orderbook.GetOrderbookBytes())
	c.SecureJSON(http.StatusOK, gin.H{"id": order.OrderID})
	return
}

func LimitOrderHandler(c *gin.Context) {
	var order models.OrderSchema
	if err := c.ShouldBindWith(&order, binding.JSON); err != nil {
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orderSide orderbook.Side
	if order.OrderSide == "buy" {
		orderSide = orderbook.Buy
	} else if order.OrderSide == "sell" {
		orderSide = orderbook.Sell
	} else {
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": "invalid side"})
		return
	}

	order.OrderType = "limit"
	order.Created = time.Now().UTC()
	order.Complete = false
	order.OrderQuantityProcessed = 0
	order.EtherQuantity = 0
	order.Fees = 0
	order.OrderID = OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)

	if !db.ValidateOrder(c.Request.Context(), order.Username, order.OrderSide, order.OrderQuantity, (order.OrderPrice*order.OrderQuantity)/global.Exchange.ETHUSD) {
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": "Could not validate order."})
		return
	}
	err := db.CreateOrder(c.Request.Context(), &order)
	if err != nil {
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	orderQuantity := decimal.NewFromFloat(order.OrderQuantity)
	orderPrice := decimal.NewFromFloat(order.OrderPrice)
	if orderQuantity.Sign() <= 0 || orderPrice.Sign() <= 0 {
		db.CancelCompleteOrder(c.Request.Context(), order.OrderID, orderbook.ErrInvalidQuantity.Error())
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": orderbook.ErrInvalidQuantity.Error()})
		return
	}

	// Attempt to process Limit Order
	quantityLeft, totalPrice, error := orderbook.ProcessLimitOrder(orderSide, order.OrderID, orderQuantity, orderPrice)
	totalPriceFloat, _ := totalPrice.Float64()
	quantityLeftFloat, _ := quantityLeft.Float64()
	log.Println(quantityLeft, totalPrice)
	if error != nil {
		db.CancelCompleteOrder(c.Request.Context(), order.OrderID, error.Error())
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}

	// If there is remaining quantity in the received order
	if quantityLeft.IsPositive() {
		// If the received order partially fulfilled orders
		if quantityLeft != orderQuantity {
			// Create a Partial Order for the remaining
			error = db.PartialLimitOrder(c.Request.Context(), order.OrderID, order.OrderQuantity-quantityLeftFloat, totalPriceFloat)
			if error != nil {
				db.CancelCompleteOrder(c.Request.Context(), order.OrderID, error.Error())
				c.SecureJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
				return
			}
		}
	} else {
		// The received order was exhausted - it fulfilled orders in the orderbook
		error = db.CompleteLimitOrder(c.Request.Context(), order.OrderID, totalPriceFloat)
		if error != nil {
			db.CancelCompleteOrder(c.Request.Context(), order.OrderID, error.Error())
			c.SecureJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
			return
		}
	}
	go orderbook.SanitizeUsersOrders(order.Username)
	go s3.UploadToS3(orderbook.GetOrderbookBytes())
	c.SecureJSON(http.StatusOK, gin.H{"id": order.OrderID})
	return
}

func CancelOrderHandler(c *gin.Context) {
	var orderID struct {
		ID string `json:"orderID" binding:"required"`
	}
	if err := c.ShouldBindWith(&orderID, binding.JSON); err != nil {
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := orderbook.CancelOrder(orderID.ID, "Order Cancelled by User"); err != nil {
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	go s3.UploadToS3(orderbook.GetOrderbookBytes())
	c.String(http.StatusOK, fmt.Sprintf("Cancelled order: %s", orderID))
	return
}
