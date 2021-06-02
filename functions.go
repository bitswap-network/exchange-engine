package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	db "v1.1-fulfiller/db"
	global "v1.1-fulfiller/global"
	ob "v1.1-fulfiller/orderbook"
)

func ProcessFull(orderlist []*ob.Order) {
	for _, order := range orderlist {
		global.WaitGroup.Add(1)
		go db.FulfillOrder(context.TODO(), order.ID(), 0, &global.WaitGroup)
	}
	return
}

func ProcessPartial(order *ob.Order, partialQuantityProcessed float64) (err error) {
	global.WaitGroup.Add(1)
	go db.PartialFulfillOrder(context.TODO(), order.ID(), partialQuantityProcessed, 0, &global.WaitGroup)
	return
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
	global.ETHUSD = price
}

func LogDepth() {
	depthMarshal, err := exchange.DepthMarshalJSON()
	if err != nil {
		log.Println(err)
		return
	}

	db.CreateDepthLog(context.TODO(), depthMarshal)
	return
}
func LogOrderbook() {
	log.Println(exchange.String())
	return
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}
