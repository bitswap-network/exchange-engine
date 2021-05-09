package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
	ob "v1.1-fulfiller/orderbook"
)

func ProcessFull(orderlist []*ob.Order) {
	for _, order := range orderlist {
		err := FulfillOrder(order.ID(), 0)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func ProcessPartial(order *ob.Order, partialQuantityProcessed decimal.Decimal) {
	pQ, _ := partialQuantityProcessed.Float64()
	err := PartialFulfillOrder(order.ID(), pQ, 0)
	if err != nil {
		log.Fatal(err)
	}
}

func OrderIDGen(orderType string, orderSide string, username string, quantity float64, created time.Time) (orderID string) {
	return fmt.Sprintf("%s-%s-%s-%v-%v", orderType, orderSide, username, quantity, created.UnixNano()/int64(time.Millisecond))
}
