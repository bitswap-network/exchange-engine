package main

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	ob "v1.1-fulfiller/orderbook"
)

func ProcessFull(orderlist []*ob.Order) (err error) {
	for _, order := range orderlist {
		wg.Add(1)
		go FulfillOrder(order.ID(), 0)
		// if err != nil {
		// 	log.Println(err)
		// 	return err
		// }
	}
	return nil
}

func ProcessPartial(order *ob.Order, partialQuantityProcessed decimal.Decimal) (err error) {
	pQ, _ := partialQuantityProcessed.Float64()
	go PartialFulfillOrder(order.ID(), pQ, 0)
	// if err != nil {
	// 	log.Println(err)
	// 	return err
	// }
	return nil
}

func OrderIDGen(orderType string, orderSide string, username string, quantity float64, created time.Time) (orderID string) {
	return fmt.Sprintf("%s-%s-%s-%v-%v", orderType, orderSide, username, quantity, created.UnixNano()/int64(time.Millisecond))
}
