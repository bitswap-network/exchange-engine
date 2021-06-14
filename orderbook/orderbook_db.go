package orderbook

import (
	"context"
	"errors"
	"log"

	"exchange-engine/db"
	"exchange-engine/global"
	"exchange-engine/s3"

	"github.com/shopspring/decimal"
)

func SanitizeUsersOrders(username string) {
	orders, err := db.GetUserOrders(context.TODO(), username)
	if err != nil {
		log.Println(err)
		return
	}
	var orderList []*Order
	for _, order := range orders {
		orderFromState := GetOrder(order.OrderID)
		log.Println(orderFromState, order.OrderID)
		if orderFromState != nil {
			orderList = append(orderList, orderFromState)
		}
	}
	Sanitize(orderList)
	return
}

func Sanitize(orders []*Order) {
	for _, order := range orders {
		log.Printf("Validating: %s\n", order.ID())
		err := validateBalance(order, false)
		if err != nil {
			log.Printf("Validation failed for: %s\n", order.ID())
			CancelOrder(order.ID(), err.Error())
		}
	}
	go s3.UploadToS3(GetOrderbookBytes())
}

// internal user balance
func validateBalance(order *Order, checkInTransaction bool) error {
	balance, err := db.GetUserBalance(context.TODO(), order.User())
	if err != nil {
		log.Println(err)
		return err
	}
	if balance.InTransaction && checkInTransaction {
		return errors.New("User in transaction while executing.")
	} else {
		totalPrice, _ := (order.Price().Mul(order.Quantity())).Float64()
		totalQuantity, _ := (order.Quantity()).Float64()
		if order.Side() == Buy {
			if totalPrice/global.Exchange.ETHUSD <= balance.Ether {
				return nil
			} else {
				return errors.New("Insufficient funds.")
			}
		} else {
			if totalQuantity <= balance.Bitclout {
				return nil
			} else {
				return errors.New("Insufficient funds.")
			}
		}
	}
}

// CancelOrder removes order with given ID from the order book
func CancelOrder(orderID string, errorString string) (*Order, error) {
	e, ok := OB.orders[orderID]
	err := db.CancelCompleteOrder(context.TODO(), orderID, errorString)
	if err != nil {
		log.Println(err.Error())
	}
	if !ok {
		return nil, ErrOrderNotExists
	}
	delete(OB.orders, orderID)
	go s3.UploadToS3(GetOrderbookBytes())
	if e.Value.(*Order).Side() == Buy {
		return OB.bids.Remove(e), nil
	}
	return OB.asks.Remove(e), nil
}

func CompleteOrder(orderID string) *Order {
	e, ok := OB.orders[orderID]
	if !ok {
		return nil
	}
	err := db.CompleteLimitOrderDirect(context.TODO(), orderID)
	if err != nil {
		go db.CancelCompleteOrder(context.TODO(), orderID, err.Error())
	}
	delete(OB.orders, orderID)
	go s3.UploadToS3(GetOrderbookBytes())
	if e.Value.(*Order).Side() == Buy {
		return OB.bids.Remove(e)
	}
	return OB.asks.Remove(e)
}

func PartialOrder(orderID string, quantityDelta decimal.Decimal) *Order {
	headOrder, ok := OB.orders[orderID]
	if !ok {
		return nil
	}
	// Fulfills an order for `quantityDelta`
	quantityDeltaFloat, _ := quantityDelta.Float64()
	err := db.PartialLimitOrderDirect(context.TODO(), orderID, quantityDeltaFloat)
	if err != nil {
		db.CancelCompleteOrder(context.TODO(), orderID, err.Error())
		CancelOrder(orderID, err.Error())
	}
	// Updates the headOrder to set the REMAINING QUANTITY to add to the OrderBook
	order := headOrder.Value.(*Order)
	partialOrder := NewOrder(orderID, order.Side(), order.Quantity().Sub(quantityDelta), order.Price(), order.Time())

	return partialOrder
}
