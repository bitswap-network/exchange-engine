package orderbook

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
)

const TEST string = "test"

//Populates the orderbook with limit sells and limit buys.
// Does NOT fulfill any orders
// func TestpopulateOrderbookWithBuyOrders(t *testing.T, OB *OrderBook, quantity decimal.Decimal) {
// 	// Create 50 Buy orders
// 	for i := 50; i <= 100; i++ {
// 		var order models.OrderSchema

// 		// Initialize the Order
// 		order.OrderSide = "buy"
// 		order.OrderType = "market"
// 		order.Created = time.Now().UTC()
// 		order.OrderID = main.OrderIDGen(order.OrderType, order.OrderSide, order.Username, order.OrderQuantity, order.Created)
// 		newOrder, err := NewOrder(order.OrderID, Buy, quantity, decimal.New(int64(i), 0), order.Created, false)
// 		if err != nil {
// 			t.Errorf("Could not create order with price %d\n"+err.Error(), i)
// 		}
// 		OB.orders[order.OrderID] = OB.bids.Append(newOrder)
// 	}
// 	if len(OB.orders) != 50 {
// 		t.Errorf("Bids size is %d, expected 50", len(OB.orders))
// 	}
// }

func TestPlaceLimitBuyOrders(t *testing.T) {
	// Create a blank orderbook (clearing the orderbook)
	Setup(true)
	quantity := decimal.New(2, 0)
	for i := 50; i < 100; i = i + 10 {
		quantityLeft, fullPrice, err := ProcessLimitOrder(Buy, fmt.Sprintf("buy-%d", i), quantity, decimal.New(int64(i), 0))
		if err != nil {
			t.Fatalf("Could not create or process order %d\n"+err.Error(), i)
		}
		if fullPrice.Cmp(decimal.New(0, 0)) != 0 || quantityLeft.Cmp(quantity) != 0 {
			t.Fatal("OrderBook fulfilled Buy orders with Buy orders (Unexpected behaviour)")
		}
	}
}

func TestPlaceLimitSellOrders(t *testing.T) {
	// Create a blank orderbook (clearing the orderbook)
	Setup(true)
	quantity := decimal.New(2, 0)

	for i := 50; i < 100; i = i + 10 {
		quantityLeft, fullPrice, err := ProcessLimitOrder(Sell, fmt.Sprintf("sell-%d", i), quantity, decimal.New(int64(i), 0))
		if err != nil {
			t.Fatalf("Could not create or process order %d\n"+err.Error(), i)
		}
		if fullPrice.Cmp(decimal.New(0, 0)) != 0 || quantityLeft.Cmp(quantity) != 0 {
			t.Fatal("OrderBook fulfilled Sell orders with Sell orders (Unexpected behaviour)")
		}
	}
}
