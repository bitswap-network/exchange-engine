package orderbook

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/shopspring/decimal"
	"v1.1-fulfiller/db"
	"v1.1-fulfiller/global"
	"v1.1-fulfiller/models"
	"v1.1-fulfiller/s3"
)

// OrderBook implements standard matching algorithm
type OrderBook struct {
	orders map[string]*list.Element // orderID -> *Order (*list.Element.Value.(*Order))

	asks *OrderSide
	bids *OrderSide
}

var OB = &OrderBook{}

/*
Initializes the orderbook

Arguments:
	blank - Whether the orderbook is empty. If true, the orderbook is retrieved from the s3 bucket.
*/
func Setup(blank bool) {
	log.Println("orderbook setup")
	if !blank {
		recoverOrderbook := s3.GetOrderbook()
		if recoverOrderbook != nil {
			log.Println("unmarshalling fetched orderbook")
			err := UnmarshalJSON(recoverOrderbook)
			if err != nil {
				log.Fatalln("Error loading fetched orderbook")
			}
			log.Println(String())
		}
	} else {
		OB = NewOrderBook()
	}
	log.Printf("orderbook setup complete\n%v", String())
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
//
// Return:
//      error        - not nil if price is less or equal 0
//      quantityLeft - More than zero if there are too few orders to process the `quantity`
//      fullPrice - The total price of the existing orders fulfilled using `quantity`. Zero if no orders are fulfilled.
func ProcessMarketOrder(side Side, quantity decimal.Decimal) (quantityLeft decimal.Decimal, fullPrice decimal.Decimal, err error) {
	if quantity.Sign() <= 0 {
		return decimal.Zero, decimal.Zero, ErrInvalidQuantity
	}
	// fullPrice = decimal.Zero
	var (
		iter          func() *OrderQueue
		sideToProcess *OrderSide
	)

	if side == Buy {
		iter = OB.asks.MinPriceQueue
		sideToProcess = OB.asks
	} else {
		iter = OB.bids.MaxPriceQueue
		sideToProcess = OB.bids
	}

	for quantity.Sign() > 0 && sideToProcess.Len() > 0 {
		bestPrice := iter()
		quantityLeft, totalPrice := processQueue(bestPrice, quantity)
		fullPrice = fullPrice.Add(totalPrice)
		quantity = quantityLeft
	}
	quantityLeft = quantity
	return
}

// ProcessLimitOrder places new order to the OrderBook
// Arguments:
//      side     - what do you want to do (ob.Sell or ob.Buy)
//      orderID  - unique order ID in depth
//      quantity - how much quantity you want to sell or buy
//      price    - no more expensive (or cheaper) this price
//      * to create new decimal number you should use decimal.New() func
//
// Return:
//      error   - not nil if quantity (or price) is less or equal 0. Or if order with given ID is exists
//      done    - not nil if your order produces ends of anoter order, this order will add to
//                the "done" slice. If your order have done too, it will be places to this array too
//      partial - not nil if your order has done but top order is not fully done. Or if your order is
//                partial done and placed to the orderbook without full quantity - partial will contain
//                your order with quantity to left
//      partialQuantityProcessed - if partial order is not nil this result contains processed quatity from partial order
func ProcessLimitOrder(side Side, orderID string, quantity, price decimal.Decimal) (quantityToTrade decimal.Decimal, totalPrice decimal.Decimal, err error) {
	if _, ok := OB.orders[orderID]; ok {
		return decimal.Zero, decimal.Zero, ErrOrderExists
	}

	if quantity.Sign() <= 0 {
		return decimal.Zero, decimal.Zero, ErrInvalidQuantity
	}

	if price.Sign() <= 0 {
		return decimal.Zero, decimal.Zero, ErrInvalidPrice
	}

	quantityToTrade = quantity
	// quantityLeft = quantityToTrade
	var (
		sideToProcess *OrderSide
		sideToAdd     *OrderSide
		comparator    func(decimal.Decimal) bool
		iter          func() *OrderQueue
	)

	if side == Buy {
		sideToAdd = OB.bids
		sideToProcess = OB.asks
		comparator = price.GreaterThanOrEqual
		iter = OB.asks.MinPriceQueue
	} else {
		sideToAdd = OB.asks
		sideToProcess = OB.bids
		comparator = price.LessThanOrEqual
		iter = OB.bids.MaxPriceQueue
	}
	bestPrice := iter()
	for quantityToTrade.Sign() > 0 && sideToProcess.Len() > 0 && comparator(bestPrice.Price()) {
		quantityToTrade, totalPrice = processQueue(bestPrice, quantityToTrade)
		bestPrice = iter()
	}

	if quantityToTrade.Sign() > 0 {
		o, err := NewOrder(orderID, side, quantityToTrade, price, time.Now().UTC(), quantity != quantityToTrade)
		if err != nil {
			log.Println(err.Error())
		} else {
			OB.orders[orderID] = sideToAdd.Append(o)
		}
	}

	return
}

func processQueue(orderQueue *OrderQueue, quantityToTrade decimal.Decimal) (quantityLeft decimal.Decimal, totalPrice decimal.Decimal) {
	quantityLeft = quantityToTrade
	for orderQueue.Len() > 0 && quantityLeft.Sign() > 0 {
		headOrderEl := orderQueue.Head()
		headOrder := headOrderEl.Value.(*Order)
		err := validateBalance(headOrder)
		if err == nil {
			//partial order
			if quantityLeft.LessThan(headOrder.Quantity()) {
				// create a new order with the remaining quantity.
				executionPrice := quantityLeft.Mul(headOrder.Price())
				executionPriceFloat, _ := executionPrice.Float64()
				totalPrice = totalPrice.Add(executionPrice)
				partial := PartialOrder(headOrder.ID(), quantityLeft, executionPriceFloat)
				orderQueue.Update(headOrderEl, partial)
				quantityLeft = decimal.Zero
			} else {
				deltaTotalPrice := headOrder.Quantity().Mul(headOrder.Price())
				deltaPriceFloat, _ := deltaTotalPrice.Float64()
				quantityLeft = quantityLeft.Sub(headOrder.Quantity())
				totalPrice = totalPrice.Add(deltaTotalPrice)
				CompleteOrder(headOrder.ID(), deltaPriceFloat)
			}
		} else {
			CancelOrder(headOrder.ID(), err.Error())
		}
	}
	return
}

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
		err := validateBalance(order)
		if err != nil {
			log.Printf("Validation failed for: %s\n", order.ID())
			CancelOrder(order.ID(), err.Error())
		}
	}
	go s3.UploadToS3(GetOrderbookBytes())
}

// internal user balance
func validateBalance(order *Order) error {
	balance, err := db.GetUserBalance(context.TODO(), order.User())
	if err != nil {
		log.Println(err)
		return err
	}
	if balance.InTransaction {
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

// Order returns order by id
func GetOrder(orderID string) *Order {
	e, ok := OB.orders[orderID]
	if !ok {
		return nil
	}
	return e.Value.(*Order)
}

// Depth returns price levels and volume at price level

// CancelOrder removes order with given ID from the order book
func CancelOrder(orderID string, errorString string) (*Order, error) {
	e, ok := OB.orders[orderID]
	if !ok {
		return nil, ErrOrderNotExists
	}
	err := db.CancelCompleteOrder(context.TODO(), orderID, errorString)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	delete(OB.orders, orderID)
	if e.Value.(*Order).Side() == Buy {
		return OB.bids.Remove(e), nil
	}
	go s3.UploadToS3(GetOrderbookBytes())
	return OB.asks.Remove(e), nil
}

func CompleteOrder(orderID string, totalPrice float64) *Order {
	e, ok := OB.orders[orderID]
	if !ok {
		return nil
	}
	err := db.CompleteLimitOrder(context.TODO(), orderID, totalPrice)
	if err != nil {
		db.CancelCompleteOrder(context.TODO(), orderID, err.Error())
	}
	delete(OB.orders, orderID)

	if e.Value.(*Order).Side() == Buy {
		return OB.bids.Remove(e)
	}
	go s3.UploadToS3(GetOrderbookBytes())
	return OB.asks.Remove(e)
}

func PartialOrder(orderID string, quantityProcessed decimal.Decimal, totalPrice float64) *Order {
	e, ok := OB.orders[orderID]
	if !ok {
		return nil
	}
	order := e.Value.(*Order)
	partialOrder, err := NewOrder(orderID, order.Side(), order.Quantity().Sub(quantityProcessed), order.Price(), time.Now().UTC(), true)
	if err != nil {
		log.Fatalln(err.Error())
	}
	quantityProcessedFloat, _ := quantityProcessed.Float64()
	err = db.PartialLimitOrder(context.TODO(), orderID, quantityProcessedFloat, totalPrice)
	if err != nil {
		db.CancelCompleteOrder(context.TODO(), orderID, err.Error())
		CancelOrder(orderID, err.Error())
	}
	return partialOrder
}

// CalculateMarketPrice returns total market price for requested quantity
// if err is not nil price returns total price of all levels in side
func CalculateMarketPrice(side Side, quantity decimal.Decimal) (price decimal.Decimal, err error) {
	price = decimal.Zero

	var (
		level *OrderQueue
		iter  func(decimal.Decimal) *OrderQueue
	)

	if side == Buy {
		level = OB.asks.MinPriceQueue()
		iter = OB.asks.GreaterThan
	} else {
		level = OB.bids.MaxPriceQueue()
		iter = OB.bids.LessThan
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
func String() string {
	return OB.asks.String() + "\r\n------------------------------------" + OB.bids.String()
}

// MarshalJSON implements json.Marshaler interface
func MarshalJSON() ([]byte, error) {
	return json.Marshal(
		&struct {
			Asks *OrderSide `json:"asks"`
			Bids *OrderSide `json:"bids"`
		}{
			Asks: OB.asks,
			Bids: OB.bids,
		},
	)
}

// func Depth() (asks, bids []*PriceLevel) {
// 	level := OB.asks.MaxPriceQueue()
// 	for level != nil {
// 		asks = append(asks, &PriceLevel{
// 			Price:    level.Price(),
// 			Quantity: level.Volume(),
// 		})
// 		level = OB.asks.LessThan(level.Price())
// 	}

// 	level = OB.bids.MaxPriceQueue()
// 	for level != nil {
// 		bids = append(bids, &PriceLevel{
// 			Price:    level.Price(),
// 			Quantity: level.Volume(),
// 		})
// 		level = OB.bids.LessThan(level.Price())
// 	}
// 	return
// }

func GetOrderbookBytes() (data []byte) {
	data, err := MarshalJSON()
	if err != nil {
		log.Println(err)
		return
	}
	return data
}

func DepthMarshalJSON() (*models.DepthSchema, error) {

	level := OB.asks.MaxPriceQueue()
	var asks, bids []*models.PriceLevel
	for level != nil {
		priceFloat, _ := level.Price().Float64()
		volumeFloat, _ := level.Volume().Float64()
		asks = append(asks, &models.PriceLevel{
			Price:    priceFloat,
			Quantity: volumeFloat,
		})
		level = OB.asks.LessThan(level.Price())
	}

	level = OB.bids.MaxPriceQueue()
	for level != nil {
		priceFloat, _ := level.Price().Float64()
		volumeFloat, _ := level.Volume().Float64()
		bids = append(bids, &models.PriceLevel{
			Price:    priceFloat,
			Quantity: volumeFloat,
		})
		level = OB.bids.LessThan(level.Price())
	}
	return &models.DepthSchema{
		TimeStamp: time.Now(),
		Asks:      asks,
		Bids:      bids,
	}, nil

}

// UnmarshalJSON implements json.Unmarshaler interface
func UnmarshalJSON(data []byte) error {
	obj := struct {
		Asks *OrderSide `json:"asks"`
		Bids *OrderSide `json:"bids"`
	}{}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	OB.asks = obj.Asks
	OB.bids = obj.Bids
	OB.orders = map[string]*list.Element{}

	for _, order := range OB.asks.Orders() {
		OB.orders[order.Value.(*Order).ID()] = order
	}

	for _, order := range OB.bids.Orders() {
		OB.orders[order.Value.(*Order).ID()] = order
	}

	return nil
}
