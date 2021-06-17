package global

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"exchange-engine/config"
	"exchange-engine/models"
)

type ExchangeRate struct {
	ETHUSD     float64
	LastUpdate int64
	FEE        float64
}

var Exchange = &ExchangeRate{}

func Setup() {
	log.Println("global setup")
	Exchange.FEE = 0.01
	SetETHUSD()
	log.Println("global setup complete")
}

func SetETHUSD() {
	apiResp := new(models.EthPriceAPI)
	if err := GetJson(fmt.Sprintf("https://api.etherscan.io/api?module=stats&action=ethprice&apikey=%s", config.UtilConfig.ETHERSCAN_KEY), apiResp); err != nil {
		log.Panic("ERROR ETH USD: ", err)
		return
	}
	price, err := strconv.ParseFloat(apiResp.Result.Ethusd, 64)
	if err != nil {
		log.Panic("ERROR PARSING FLOAT ETH USD: ", err)
		return
	}
	Exchange.LastUpdate = time.Now().UnixNano() / int64(time.Millisecond)
	Exchange.ETHUSD = price
}
func GetJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if err = json.NewDecoder(r.Body).Decode(target); err != nil {
		return err
	}
	return nil
}

func PostJson(url string, data []byte, target interface{}) error {
	r, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if err = json.NewDecoder(r.Body).Decode(target); err != nil {
		return err
	}
	return nil
}
