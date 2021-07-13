package fireeye

import (
	"context"
	"encoding/json"
	"exchange-engine/config"
	"exchange-engine/db"
	"exchange-engine/global"
	"exchange-engine/models"
	"fmt"
	"log"
)

func GetMainWalletBalance(ctx context.Context) (*models.GetWalletBalanceResponse, error) {
	wallet, err := db.GetMainWallet(ctx)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	getWalletBalanceMap := models.GetWalletBalanceBody{wallet.KeyInfo.Bitclout.PublicKeyBase58Check, BitcloutConfirmations}
	getWalletBalanceReqBody, err := json.Marshal(getWalletBalanceMap)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	getWalletBalanceResp := new(models.GetWalletBalanceResponse)
	if err := global.PostJson(fmt.Sprintf("%s/api/v1/balance", config.BITCLOUT_NODEURL), getWalletBalanceReqBody, getWalletBalanceResp); err != nil {
		log.Println("ERROR getWalletBalanceResp: ", err)
		return nil, err
	}
	return getWalletBalanceResp, nil
}

func GetPoolsBalance(ctx context.Context) (balanceWei uint64, err error) {
	pools, err := db.GetAllPools(ctx)
	balanceWei = 0
	for _, pool := range pools {
		balanceWei += pool.Balance.ETH
	}
	return
}
