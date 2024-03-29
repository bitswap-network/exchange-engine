package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	global "exchange-engine/global"
	ob "exchange-engine/orderbook"

	unitTest "github.com/Valiben/gin_unit_test"
	utils "github.com/Valiben/gin_unit_test/utils"
	"github.com/shopspring/decimal"
)

func init() {
	router := RouterSetup()
	unitTest.SetRouter(router)
	newLog := log.New(os.Stdout, "", log.Llongfile|log.Ldate|log.Ltime)
	unitTest.SetLog(newLog)
}

func TestRootRoute(t *testing.T) {
	resp, err := unitTest.TestOrdinaryHandler(utils.GET, "/", utils.JSON, nil)
	if err != nil {
		t.Errorf("Root Test Error: %v\n", err)
		return
	}
	if string(resp) != "Bitswap Exchange Manager" {
		t.Errorf("Unexpected Response: %v\n", string(resp))
		return
	}
}

func TestETHUSDRoute(t *testing.T) {
	respBody := struct {
		Result float64 `json:"result"`
	}{}
	err := unitTest.TestHandlerUnMarshalResp(utils.GET, "/ethusd", utils.JSON, nil, &respBody)
	if err != nil {
		t.Errorf("ETHUSD Test Error: %v\n", err)
		return
	}
	if respBody.Result != global.Exchange.ETHUSD {
		t.Errorf("Unexpected Response: %v\n", respBody)
		return
	}
}

func TestMarketPriceRoute(t *testing.T) {
	var testQuantity int64 = 10
	respBody := struct {
		Quantity float64 `json:"quantity"`
		Price    float64 `json:"price"`
		Side     string  `json:"side"`
	}{}
	log.Println(decimal.NewFromInt(testQuantity))
	priceBuy, opErr := ob.CalculateMarketPrice(ob.Buy, decimal.NewFromInt(testQuantity))
	if opErr != nil {
		t.Errorf("Market Price Test Error: %v\n", opErr)
		return
	}
	priceSell, opErr := ob.CalculateMarketPrice(ob.Sell, decimal.NewFromInt(testQuantity))
	if opErr != nil {
		t.Errorf("Market Price Test Error: %v\n", opErr)
		return
	}
	priceBuyFloat, _ := priceBuy.Float64()
	priceSellFloat, _ := priceSell.Float64()

	url := fmt.Sprintf("/market-price/buy/%v", testQuantity)
	marshalErr := unitTest.TestHandlerUnMarshalResp(utils.GET, url, utils.JSON, nil, &respBody)
	if marshalErr != nil {
		t.Errorf("Root Test Error: %v\n", marshalErr)
		return
	}
	if respBody.Price != priceBuyFloat {
		t.Errorf("Unexpected Response: %v\n", respBody)
		t.Errorf("Expected Response: %v\n", priceBuyFloat)
		return
	}
	respBody = struct {
		Quantity float64 `json:"quantity"`
		Price    float64 `json:"price"`
		Side     string  `json:"side"`
	}{}
	url = fmt.Sprintf("/market-price/sell/%v", testQuantity)
	marshalErr = unitTest.TestHandlerUnMarshalResp(utils.GET, url, utils.JSON, nil, &respBody)
	if marshalErr != nil {
		t.Errorf("Root Test Error: %v\n", marshalErr)
		return
	}
	if respBody.Price != priceSellFloat {
		t.Errorf("Unexpected Response: %v\n", respBody)
		t.Errorf("Expected Response: %v\n", priceSellFloat)
		return
	}
}
