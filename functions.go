package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"v1.1-fulfiller/db"
	"v1.1-fulfiller/orderbook"
)

func ProcessFull(orderlist []*orderbook.Order, execPrice float64) {
	for _, order := range orderlist {
		db.FulfillOrder(context.TODO(), order.ID(), 0, execPrice)
		// go s3.UploadToS3(orderbook.GetOrderbookBytes())
	}
}

func ProcessPartial(order *orderbook.Order, partialQuantityProcessed float64, execPrice float64) {
	db.PartialFulfillOrder(context.TODO(), order.ID(), partialQuantityProcessed, 0, execPrice)
	// go s3.UploadToS3(orderbook.GetOrderbookBytes())
}

func OrderIDGen(orderType string, orderSide string, username string, quantity float64, created time.Time) (orderID string) {
	return fmt.Sprintf("%s-%s-%s-%v-%v", orderType, orderSide, username, quantity, created.UnixNano()/int64(time.Millisecond))
}

func LogDepth() {
	depthMarshal, err := orderbook.DepthMarshalJSON()
	if err != nil {
		log.Println(err)
	}
	db.CreateDepthLog(context.TODO(), depthMarshal)
}
