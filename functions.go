package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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

type EthPriceAPI struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Result  EthPriceAPIResult `json:"result"`
}
type EthPriceAPIResult struct {
	Ethbtc           string `json:"ethbtc"`
	Ethbtc_timestamp string `json:"ethbtc_timestamp"`
	Ethusd           string `json:"ethusd"`
	Ethusd_timestamp string `json:"ethusd_timestamp"`
}

func SetETHUSD() {
	apiResp := new(EthPriceAPI)
	getJson(fmt.Sprintf("https://api.etherscan.io/api?module=stats&action=ethprice&apikey=%s", os.Getenv("ETHERSCAN_KEY")), apiResp)
	price, err := strconv.ParseFloat(apiResp.Result.Ethusd, 64)
	if err != nil {
		log.Println(err)
	}
	ETHUSD = price
	fmt.Println(ETHUSD)
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}
