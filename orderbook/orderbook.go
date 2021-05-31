package orderbook

import (
	"container/list"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/shopspring/decimal"
	db "v1.1-fulfiller/db"
	global "v1.1-fulfiller/global"
	model "v1.1-fulfiller/models"
	s3 "v1.1-fulfiller/s3"
)

// OrderBook implements standard matching algorithm
type OrderBook struct {
	orders map[string]*list.Element // orderID -> *Order (*list.Element.Value.(*Order))

	asks *OrderSide
	bids *OrderSide
}

// NewOrderBook creates Orderbook object
func NewOrderBook() *OrderBook {
	return &OrderBook{
		orders: map[string]*list.Element{},
		bids:   NewOrderSide(),
		asks:   NewOrderSide(),
	}
}

// PriceLevel contains price and volume in depth
type PriceLevel struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
}

// ProcessMarketOrder immediately gets definite quantity from the order book with market price
// Arguments:
//      side     - what do you want to do (ob.Sell or ob.Buy)
//      quantity - how much quantity you want to sell or buy
//      * to create new decimal number you should use decimal.New() func
//        read more at https://github.com/shopspring/decimal
// Return:
//      error        - not nil if price is less or equal 0
//      done         - not nil if your market order produces ends of anoter orders, this order will add to
//                     the "done" slice
//      partial      - not nil if your order has done but top order is not fully done
//      partialQuantityProcessed - if partial order is not nil this result contains processed quatity from partial order
//      quantityLeft - more than zero if it is not enought orders to process all quantity
func (ob *OrderBook) ProcessMarketOrder(side Side, quantity decimal.Decimal) (done []*Order, partial *Order, partialQuantityProcessed, quantityLeft decimal.Decimal, fullPrice decimal.Decimal, err error) {
	if quantity.Sign() <= 0 {
		return nil, nil, decimal.Zero, decimal.Zero, decimal.Zero, ErrInvalidQuantity
	}
	// fullPrice = decimal.Zero
	var (
		iter          func() *OrderQueue
		sideToProcess *OrderSide
	)

	if side == Buy {
		iter = ob.asks.MinPriceQueue
		sideToProcess = ob.asks
	} else {
		iter = ob.bids.MaxPriceQueue
		sideToProcess = ob.bids
	}

	for quantity.Sign() > 0 && sideToProcess.Len() > 0 {
		bestPrice := iter()
		ordersDone, partialDone, partialProcessed, quantityLeft, totalPrice := ob.processQueue(bestPrice, quantity)
		done = append(done, ordersDone...)
		partial = partialDone
		fullPrice = fullPrice.Add(totalPrice)
		partialQuantityProcessed = partialProcessed
		quantity = quantityLeft
	}

	quantityLeft = quantity
	if partial != nil {
		ob.Sanitize(append(done, partial))
	} else {
		ob.Sanitize(done)
	}
	return
}

// ProcessLimitOrder places new order to the OrderBook
// Arguments:
//      side     - what do you want to do (ob.Sell or ob.Buy)
//      orderID  - unique order ID in depth
//      quantity - how much quantity you want to sell or buy
//      price    - no more expensive (or cheaper) this price
//      * to create new decimal number you should use decimal.New() func
//        read more at https://github.com/shopspring/decimal
// Return:
//      error   - not nil if quantity (or price) is less or equal 0. Or if order with given ID is exists
//      done    - not nil if your order produces ends of anoter order, this order will add to
//                the "done" slice. If your order have done too, it will be places to this array too
//      partial - not nil if your order has done but top order is not fully done. Or if your order is
//                partial done and placed to the orderbook without full quantity - partial will contain
//                your order with quantity to left
//      partialQuantityProcessed - if partial order is not nil this result contains processed quatity from partial order
func (ob *OrderBook) ProcessLimitOrder(side Side, orderID string, quantity, price decimal.Decimal) (done []*Order, partial *Order, partialQuantityProcessed decimal.Decimal, err error) {
	if _, ok := ob.orders[orderID]; ok {
		return nil, nil, decimal.Zero, ErrOrderExists
	}

	if quantity.Sign() <= 0 {
		return nil, nil, decimal.Zero, ErrInvalidQuantity
	}

	if price.Sign() <= 0 {
		return nil, nil, decimal.Zero, ErrInvalidPrice
	}

	quantityToTrade := quantity
	var (
		sideToProcess *OrderSide
		sideToAdd     *OrderSide
		comparator    func(decimal.Decimal) bool
		iter          func() *OrderQueue
	)

	if side == Buy {
		sideToAdd = ob.bids
		sideToProcess = ob.asks
		comparator = price.GreaterThanOrEqual
		iter = ob.asks.MinPriceQueue
	} else {
		sideToAdd = ob.asks
		sideToProcess = ob.bids
		comparator = price.LessThanOrEqual
		iter = ob.bids.MaxPriceQueue
	}

	bestPrice := iter()
	for quantityToTrade.Sign() > 0 && sideToProcess.Len() > 0 && comparator(bestPrice.Price()) {
		ordersDone, partialDone, partialQty, quantityLeft, _ := ob.processQueue(bestPrice, quantityToTrade)
		done = append(done, ordersDone...)
		partial = partialDone
		partialQuantityProcessed = partialQty
		quantityToTrade = quantityLeft
		bestPrice = iter()
	}

	if quantityToTrade.Sign() > 0 {
		o := NewOrder(orderID, side, quantityToTrade, price, time.Now().UTC())
		if len(done) > 0 {
			partialQuantityProcessed = quantity.Sub(quantityToTrade)
			partial = o
		}
		ob.orders[orderID] = sideToAdd.Append(o)
	} else {
		totalQuantity := decimal.Zero
		totalPrice := decimal.Zero

		for _, order := range done {
			totalQuantity = totalQuantity.Add(order.Quantity())
			totalPrice = totalPrice.Add(order.Price().Mul(order.Quantity()))
		}

		if partialQuantityProcessed.Sign() > 0 {
			totalQuantity = totalQuantity.Add(partialQuantityProcessed)
			totalPrice = totalPrice.Add(partial.Price().Mul(partialQuantityProcessed))
		}
		done = append(done, NewOrder(orderID, side, quantity, totalPrice.Div(totalQuantity), time.Now().UTC()))
	}
	if partial != nil {
		ob.Sanitize(append(done, partial))
	} else {
		ob.Sanitize(done)
	}

	return
}

func (ob *OrderBook) processQueue(orderQueue *OrderQueue, quantityToTrade decimal.Decimal) (done []*Order, partial *Order, partialQuantityProcessed decimal.Decimal, quantityLeft decimal.Decimal, totalPrice decimal.Decimal) {
	// totalPrice = decimal.Zero
	quantityLeft = quantityToTrade

	for orderQueue.Len() > 0 && quantityLeft.Sign() > 0 {
		headOrderEl := orderQueue.Head()
		headOrder := headOrderEl.Value.(*Order)
		if ob.validateBalance(headOrder) {
			log.Println("validation passed")
			if quantityLeft.LessThan(headOrder.Quantity()) {
				partial = NewOrder(headOrder.ID(), headOrder.Side(), headOrder.Quantity().Sub(quantityLeft), headOrder.Price(), headOrder.Time())
				partialQuantityProcessed = quantityLeft
				totalPrice = totalPrice.Add(partialQuantityProcessed.Mul(headOrder.Price()))
				orderQueue.Update(headOrderEl, partial)
				quantityLeft = decimal.Zero
			} else {
				quantityLeft = quantityLeft.Sub(headOrder.Quantity())
				totalPrice = totalPrice.Add(headOrder.Quantity().Mul(headOrder.Price()))
				done = append(done, ob.CancelOrder(headOrder.ID()))
			}
		} else {
			log.Println("validation failed")
			global.Wg.Add(1)
			go db.CancelCompleteOrder(context.TODO(), headOrder.ID(), "Order cancelled due to insufficient funds.", &global.Wg)
			ob.CancelOrder(headOrder.ID())
		}
	}
	return
}

//change to only validate users associated with orders
func (ob *OrderBook) Sanitize(orders []*Order) {
	for _, order := range orders {
		log.Printf("Validating: %s\n", order.ID())
		if !ob.validateBalance(order) {
			log.Printf("Validation failed for: %s\n", order.ID())
			global.Wg.Add(1)
			go db.CancelCompleteOrder(context.TODO(), order.ID(), "Order cancelled during sanitization due to insufficient funds.", &global.Wg)
			ob.CancelOrder(order.ID())
		}
	}
	go s3.UploadToS3(ob.GetOrderbookBytes())
}

// internal user balance
func (ob *OrderBook) validateBalance(order *Order) bool {
	balance, err := db.GetUserBalanceFromOrder(context.TODO(), order.ID())
	//IMPORTANT: must change - only for debug
	if err != nil {
		log.Println(err)
		return false
	}

	totalPrice, _ := (order.Price().Mul(order.Quantity())).Float64()
	totalQuantity, _ := (order.Quantity()).Float64()
	if order.Side() == Buy {
		return totalPrice/global.ETHUSD <= balance.Ether
	} else {
		return totalQuantity <= balance.Bitclout
	}
}

// Order returns order by id
func (ob *OrderBook) Order(orderID string) *Order {
	e, ok := ob.orders[orderID]
	if !ok {
		return nil
	}

	return e.Value.(*Order)
}

// Depth returns price levels and volume at price level

// CancelOrder removes order with given ID from the order book
func (ob *OrderBook) CancelOrder(orderID string) *Order {
	e, ok := ob.orders[orderID]
	if !ok {
		return nil
	}

	delete(ob.orders, orderID)

	if e.Value.(*Order).Side() == Buy {
		return ob.bids.Remove(e)
	}

	return ob.asks.Remove(e)
}

// CalculateMarketPrice returns total market price for requested quantity
// if err is not nil price returns total price of all levels in side
func (ob *OrderBook) CalculateMarketPrice(side Side, quantity decimal.Decimal) (price decimal.Decimal, err error) {
	price = decimal.Zero

	var (
		level *OrderQueue
		iter  func(decimal.Decimal) *OrderQueue
	)

	if side == Buy {
		level = ob.asks.MinPriceQueue()
		iter = ob.asks.GreaterThan
	} else {
		level = ob.bids.MaxPriceQueue()
		iter = ob.bids.LessThan
	}

	for quantity.Sign() > 0 && level != nil {
		levelVolume := level.Volume()
		levelPrice := level.Price()
		if quantity.GreaterThanOrEqual(levelVolume) {
			price = price.Add(levelPrice.Mul(levelVolume))
			quantity = quantity.Sub(levelVolume)
			level = iter(levelPrice)
		} else {
			price = price.Add(levelPrice.Mul(quantity))
			quantity = decimal.Zero
		}
	}

	if quantity.Sign() > 0 {
		err = ErrInsufficientQuantity
	}

	return
}

// String implements fmt.Stringer interface
func (ob *OrderBook) String() string {
	return ob.asks.String() + "\r\n------------------------------------" + ob.bids.String()
}

// MarshalJSON implements json.Marshaler interface
func (ob *OrderBook) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		&struct {
			Asks *OrderSide `json:"asks"`
			Bids *OrderSide `json:"bids"`
		}{
			Asks: ob.asks,
			Bids: ob.bids,
		},
	)
}

func (ob *OrderBook) Depth() (asks, bids []*PriceLevel) {
	level := ob.asks.MaxPriceQueue()
	for level != nil {
		asks = append(asks, &PriceLevel{
			Price:    level.Price(),
			Quantity: level.Volume(),
		})
		level = ob.asks.LessThan(level.Price())
	}

	level = ob.bids.MaxPriceQueue()
	for level != nil {
		bids = append(bids, &PriceLevel{
			Price:    level.Price(),
			Quantity: level.Volume(),
		})
		level = ob.bids.LessThan(level.Price())
	}
	return
}

func (ob *OrderBook) GetOrderbookBytes() (data []byte) {
	data, err := ob.MarshalJSON()
	if err != nil {
		log.Println(err)
		return
	}
	return data
}

func (ob *OrderBook) DepthMarshalJSON() (*model.DepthSchema, error) {
	buyMarketPrice, err := ob.CalculateMarketPrice(Buy, decimal.NewFromInt(1))
	if err != nil {
		return nil, err
	}
	sellMarketPrice, err := ob.CalculateMarketPrice(Sell, decimal.NewFromInt(1))
	if err != nil {
		return nil, err
	}
	buyMarketPriceFloat, _ := buyMarketPrice.Float64()
	sellMarketPriceFloat, _ := sellMarketPrice.Float64()
	level := ob.asks.MaxPriceQueue()
	var asks, bids []*model.PriceLevel
	for level != nil {
		priceFloat, _ := level.Price().Float64()
		volumeFloat, _ := level.Volume().Float64()
		asks = append(asks, &model.PriceLevel{
			Price:    priceFloat,
			Quantity: volumeFloat,
		})
		level = ob.asks.LessThan(level.Price())
	}

	level = ob.bids.MaxPriceQueue()
	for level != nil {
		priceFloat, _ := level.Price().Float64()
		volumeFloat, _ := level.Volume().Float64()
		bids = append(bids, &model.PriceLevel{
			Price:    priceFloat,
			Quantity: volumeFloat,
		})
		level = ob.bids.LessThan(level.Price())
	}
	return &model.DepthSchema{
		TimeStamp:  time.Now(),
		MarketBuy:  buyMarketPriceFloat,
		MarketSell: sellMarketPriceFloat,
		Asks:       asks,
		Bids:       bids,
	}, nil

}

// UnmarshalJSON implements json.Unmarshaler interface
func (ob *OrderBook) UnmarshalJSON(data []byte) error {
	obj := struct {
		Asks *OrderSide `json:"asks"`
		Bids *OrderSide `json:"bids"`
	}{}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	ob.asks = obj.Asks
	ob.bids = obj.Bids
	ob.orders = map[string]*list.Element{}

	for _, order := range ob.asks.Orders() {
		ob.orders[order.Value.(*Order).ID()] = order
	}

	for _, order := range ob.bids.Orders() {
		ob.orders[order.Value.(*Order).ID()] = order
	}

	return nil
}
