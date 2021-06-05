package global

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"v1.1-fulfiller/config"
	"v1.1-fulfiller/models"
)

type ExchangeRate struct {
	ETHUSD     float64
	LastUpdate int64
	FEE        float64
}

var Exchange = &ExchangeRate{}

var WaitGroup sync.WaitGroup

func Setup() {
	log.Println("global setup")
	Exchange.FEE = 0.02
	SetETHUSD()
	log.Println("global setup complete")
}

func SetETHUSD() {
	log.Println("getting ethusd")
	apiResp := new(models.EthPriceAPI)
	getJson(fmt.Sprintf("https://api.etherscan.io/api?module=stats&action=ethprice&apikey=%s", config.UtilConfig.ETHERSCAN_KEY), apiResp)
	price, err := strconv.ParseFloat(apiResp.Result.Ethusd, 64)
	if err != nil {
		log.Println(err.Error())
	}
	Exchange.LastUpdate = time.Now().UnixNano() / int64(time.Millisecond)
	Exchange.ETHUSD = price
	log.Printf("Current ETHUSD price: %f", Exchange.ETHUSD)
}
func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	json.NewDecoder(r.Body).Decode(target)
	defer r.Body.Close()
	return nil
}