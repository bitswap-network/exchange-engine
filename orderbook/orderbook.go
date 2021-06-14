package orderbook

import (
	"container/list"
	"encoding/json"
	"log"
	"time"

	"exchange-engine/models"
	"exchange-engine/s3"

	"github.com/shopspring/decimal"
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
	blank - Whether the orderbook is empty. If false, the orderbook is retrieved from the s3 bucket.
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
	quantityToTrade := quantity
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

	for quantityToTrade.Sign() > 0 && sideToProcess.Len() > 0 {
		bestPrice := iter()
		quantityLeft, totalPrice := processQueue(bestPrice, quantityToTrade)
		fullPrice = fullPrice.Add(totalPrice)
		quantityToTrade = quantityLeft
	}
	quantityLeft = quantityToTrade
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
func ProcessLimitOrder(side Side, orderID string, quantity, price decimal.Decimal) (quantityToTrade decimal.Decimal, fullPrice decimal.Decimal, err error) {
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
		quantityLeft, totalPrice := processQueue(bestPrice, quantityToTrade)
		fullPrice = fullPrice.Add(totalPrice)
		quantityToTrade = quantityLeft
		bestPrice = iter()
	}

	//If the given order has exhausted the price depth
	if quantityToTrade.Sign() > 0 {
		o := NewOrder(orderID, side, quantityToTrade, price, time.Now().UTC())
		OB.orders[orderID] = sideToAdd.Append(o)
	}

	return
}

func processQueue(orderQueue *OrderQueue, quantityToTrade decimal.Decimal) (quantityLeft decimal.Decimal, totalPrice decimal.Decimal) {
	totalPrice = decimal.Zero
	quantityLeft = quantityToTrade
	for orderQueue.Len() > 0 && quantityLeft.Sign() > 0 {
		headOrderEl := orderQueue.Head()
		headOrder := headOrderEl.Value.(*Order)
		err := validateBalance(headOrder, true)
		if err == nil {
			//partial order
			if quantityLeft.LessThan(headOrder.Quantity()) {
				// create a new order with the remaining quantity.
				totalPrice = totalPrice.Add(quantityLeft.Mul(headOrder.Price()))
				partial := PartialOrder(headOrder.ID(), quantityLeft)
				log.Printf("Partial price: %s %s", totalPrice.String(), quantityLeft.Mul(headOrder.Price()).String())
				orderQueue.Update(headOrderEl, partial)
				quantityLeft = decimal.Zero
			} else {
				//full order
				quantityLeft = quantityLeft.Sub(headOrder.Quantity())
				totalPrice = totalPrice.Add(headOrder.Quantity().Mul(headOrder.Price()))
				log.Printf("Complete price: %s %s", totalPrice.String(), headOrder.Quantity().Mul(headOrder.Price()).String())
				CompleteOrder(headOrder.ID())
			}
		} else {
			if err = CancelOrder(headOrder.ID(), err.Error()); err != nil {
				log.Println(err.Error())
			}
		}
	}
	return
}

// Order returns order by id
func GetOrder(orderID string) *Order {
	e, ok := OB.orders[orderID]
	if !ok {
		return nil
	}
	return e.Value.(*Order)
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
