package orderbook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"v1.1-fulfiller/db"
	"v1.1-fulfiller/models"
)

// Order strores information about request
type Order struct {
	side      Side
	id        string
	timestamp time.Time
	quantity  decimal.Decimal
	price     decimal.Decimal
}

/*
NewOrder creates new constant object Order
Arguments:
	orderID - The ID of the order to create
	side - Whether the order is an `ob.Buy` or an `ob.Sell`
	price - The price at which the order is created
	timestamp - The time at which the order was created
	update - Whether to update the database with the order. If update is false, no database calls are created
*/
func NewOrder(orderID string, side Side, quantity, price decimal.Decimal, timestamp time.Time, update bool) (*Order, error) {

	if update {
		quantFloat, _ := quantity.Float64()
		priceFloat, _ := price.Float64()

		dbOrder := &models.OrderSchema{
			OrderID:                orderID,
			OrderSide:              side.String(),
			OrderQuantityProcessed: quantFloat,
			OrderPrice:             priceFloat,
			Created:                timestamp,
		}
		err := db.UpdateOrder(context.TODO(), dbOrder)
		if err != nil {
			log.Fatalln(err.Error())
			return nil, err
		}
	}

	return &Order{
		id:        orderID,
		side:      side,
		quantity:  quantity,
		price:     price,
		timestamp: timestamp,
	}, nil
}

// func UpdateOrder(orderID string) (*Order, error) {
// 	err := db.Update()
// }

func (o *Order) User() string {
	s := strings.Split(o.id, "-")
	return s[2]
}

// ID returns orderID field copy
func (o *Order) ID() string {
	return o.id
}

// Side returns side of the order
func (o *Order) Side() Side {
	return o.side
}

// Quantity returns quantity field copy
func (o *Order) Quantity() decimal.Decimal {
	return o.quantity
}

// Price returns price field copy
func (o *Order) Price() decimal.Decimal {
	return o.price
}

// Time returns timestamp field copy
func (o *Order) Time() time.Time {
	return o.timestamp
}

// String implements Stringer interface
func (o *Order) String() string {
	return fmt.Sprintf("\n\"%s\":\n\tside: %s\n\tquantity: %s\n\tprice: %s\n\ttime: %s\n", o.ID(), o.Side(), o.Quantity(), o.Price(), o.Time())
}

// MarshalJSON implements json.Marshaler interface
func (o *Order) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		&struct {
			S         Side            `json:"side"`
			ID        string          `json:"id"`
			Timestamp time.Time       `json:"timestamp"`
			Quantity  decimal.Decimal `json:"quantity"`
			Price     decimal.Decimal `json:"price"`
		}{
			S:         o.Side(),
			ID:        o.ID(),
			Timestamp: o.Time(),
			Quantity:  o.Quantity(),
			Price:     o.Price(),
		},
	)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (o *Order) UnmarshalJSON(data []byte) error {
	obj := struct {
		S         Side            `json:"side"`
		ID        string          `json:"id"`
		Timestamp time.Time       `json:"timestamp"`
		Quantity  decimal.Decimal `json:"quantity"`
		Price     decimal.Decimal `json:"price"`
	}{}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	o.side = obj.S
	o.id = obj.ID
	o.timestamp = obj.Timestamp
	o.quantity = obj.Quantity
	o.price = obj.Price
	return nil
}
