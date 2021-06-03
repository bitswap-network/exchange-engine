package main

import (
	"context"
	"fmt"
	"log"
	"time"

	db "v1.1-fulfiller/db"
	ob "v1.1-fulfiller/orderbook"
)

func ProcessFull(orderlist []*ob.Order) {
	for _, order := range orderlist {
		go db.FulfillOrder(context.TODO(), order.ID(), 0)
	}
}

func ProcessPartial(order *ob.Order, partialQuantityProcessed float64) {
	go db.PartialFulfillOrder(context.TODO(), order.ID(), partialQuantityProcessed, 0)
}

func OrderIDGen(orderType string, orderSide string, username string, quantity float64, created time.Time) (orderID string) {
	return fmt.Sprintf("%s-%s-%s-%v-%v", orderType, orderSide, username, quantity, created.UnixNano()/int64(time.Millisecond))
}

func LogDepth() {
	depthMarshal, err := ob.DepthMarshalJSON()
	if err != nil {
		log.Println(err)
	}
	db.CreateDepthLog(context.TODO(), depthMarshal)
}

func LogOrderbook() {
	log.Println(ob.String())
}
