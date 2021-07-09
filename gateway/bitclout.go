package gateway

import (
	"context"
	"encoding/json"
	"exchange-engine/db"
	"exchange-engine/fireeye"
	"exchange-engine/global"
	"exchange-engine/models"
	"log"
	"sync"
	"time"
)

func QueryWallets(ctx context.Context) {
	start := time.Now()
	defer func() {
		log.Println("Execution Time: ", time.Since(start))
	}()
	wg := sync.WaitGroup{}
	wallets, err := db.GetAllWallets(ctx)
	if err != nil {
		log.Panic(err)
		return
	}
	for _, wallet := range wallets {
		wg.Add(1)
		time.Sleep(50 * time.Millisecond) // to prevent api from getting overwhelmed
		go func(wallet *models.WalletSchema) {
			walletBalanceNanos, err := GetWalletBalance(wallet)
			if err != nil {
				log.Println(err)
			}
			if walletBalanceNanos-wallet.Fees.Bitclout > 1000 {

			}
			wg.Done()
		}(wallet)
	}
	wg.Wait()
}

func GetWalletBalance(wallet *models.WalletSchema) (balance uint64, err error) {
	getWalletBalanceMap := models.GetWalletBalanceBody{wallet.KeyInfo.Bitclout.PublicKeyBase58Check, fireeye.BitcloutConfirmations}
	getWalletBalanceReqBody, _ := json.Marshal(getWalletBalanceMap)
	if err != nil {
		log.Println(err)
	}
	getWalletBalanceResp := new(models.GetWalletBalanceResponse)
	err = global.PostJson("http://node.bitswap.network/api/v1/balance", getWalletBalanceReqBody, getWalletBalanceResp)

	balance = getWalletBalanceResp.ConfirmedBalanceNanos
	return
}

func CreateDeposit(wallet *models.WalletSchema) (balance uint64, err error) {
	getWalletBalanceMap := models.GetWalletBalanceBody{wallet.KeyInfo.Bitclout.PublicKeyBase58Check, fireeye.BitcloutConfirmations}
	getWalletBalanceReqBody, _ := json.Marshal(getWalletBalanceMap)
	if err != nil {
		log.Println(err)
	}
	getWalletBalanceResp := new(models.GetWalletBalanceResponse)
	err = global.PostJson("http://node.bitswap.network/api/v1/balance", getWalletBalanceReqBody, getWalletBalanceResp)

	balance = getWalletBalanceResp.ConfirmedBalanceNanos
	return
}
