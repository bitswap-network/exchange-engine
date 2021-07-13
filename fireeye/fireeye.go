package fireeye

import (
	"context"
	"log"
	"math/big"

	"exchange-engine/config"
	"exchange-engine/db"
	"exchange-engine/global"
	"exchange-engine/models"
)

func SyncStatus(ctx context.Context) {

	TotalAccount, err := db.GetTotalBalances(ctx)
	checkErr(err)
	totalAccountBitclout, err := global.FromNanosBig(TotalAccount.Bitclout)
	checkErr(err)
	totalAccountEther, err := global.FromWeiBig(TotalAccount.Ether)
	checkErr(err)

	TotalFees, err := db.GetOrderFees(ctx)
	checkErr(err)

	totalFeesBitclout, err := global.FromNanosBig(TotalFees.Bitclout)
	checkErr(err)
	totalFeesEther, err := global.FromWeiBig(TotalFees.Ether)
	checkErr(err)
	walletBalance, err := GetMainWalletBalance(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	walletBitcloutBalanceNanos := walletBalance.ConfirmedBalanceNanos + walletBalance.UnconfirmedBalanceNanos
	walletBitcloutBalance := global.FromNanos(walletBitcloutBalanceNanos)

	walletEtherBalanceWei, err := GetPoolsBalance(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	walletEtherBalance := global.FromWei(walletEtherBalanceWei)

	FireEye.WalletBalance = models.CurrencyAmounts{float64(walletBitcloutBalanceNanos), float64(walletEtherBalanceWei)}
	FireEye.TotalAccount = *TotalAccount
	FireEye.TotalFees = *TotalFees

	// var (
	// 	bitcloutSync = totalAccountBitclout + totalFeesBitclout + config.Wallet.InitBcltTolerance
	// 	etherSync    = totalAccountEther + totalFeesEther + config.Wallet.InitEthTolerance
	// )

	// bitcloutDeviation := math.Abs((bitcloutSync / walletBitcloutBalance) - 1)
	// etherDeviation := math.Abs((etherSync / walletEtherBalance) - 1)

	errMsg := false
	bitcloutTotal := new(big.Float)
	etherTotal := new(big.Float)
	bitcloutDeviation := new(big.Float)
	etherDeviation := new(big.Float)
	etherTotal = etherTotal.Add(totalAccountEther, totalFeesEther)
	etherTotal = etherTotal.Add(etherTotal, big.NewFloat(config.Wallet.InitEthTolerance))
	bitcloutTotal = bitcloutTotal.Add(totalAccountBitclout, totalFeesBitclout)
	bitcloutTotal = bitcloutTotal.Add(bitcloutTotal, big.NewFloat(config.Wallet.InitBcltTolerance))
	/*
		if bitclout deviation >= 0.5 -> Out of sync
		if <0.5 && >-1 -> OK
	*/
	switch bitcloutDeviation := bitcloutDeviation.Sub(bitcloutTotal, big.NewFloat(walletBitcloutBalance)); {
	case bitcloutDeviation.Cmp(big.NewFloat(0.5)) >= 0: // Cmp returns +1 if bitcloutDeviation > 0.5 and 0 if bitcloutDeviation = 0.5
		FireEye.Code = 32
		FireEye.Message = "Bitclout balance out of sync."
		errMsg = true
	case bitcloutDeviation.Cmp(big.NewFloat(-1)) <= 0: // Cmp returns -1 or 0 if bitcloutDeviation <= -1
		FireEye.Code = 35
		FireEye.Message = "Unexpected values bitclout."
		errMsg = true
	default:
		FireEye.Code = 0
		FireEye.Message = "OK"
	}
	/*
		if ether deviation >= 0.05 -> Out of sync
		if <0.1 && >-1 -> OK
	*/
	switch etherDeviation := etherDeviation.Sub(etherTotal, big.NewFloat(walletEtherBalance)); {
	case etherDeviation.Cmp(big.NewFloat(0.05)) >= 0: // Cmp returns +1 if etherDeviation > 0.05 and 0 if bitcloutDeviation = 0.05
		if FireEye.Code == 32 {
			FireEye.Code = 33
			FireEye.Message = "Bitclout and Ether balances out of sync."
		} else {
			FireEye.Code = 31
			FireEye.Message = "Ether balance out of sync."
		}
		errMsg = true
	case etherDeviation.Cmp(big.NewFloat(-0.05)) <= 0: // Cmp returns -1 or 0 if etherDeviation <= -0.05
		if FireEye.Code == 35 {
			FireEye.Code = 36
			FireEye.Message = "Unexpected values ether & bitclout."
		} else {
			FireEye.Code = 34
			FireEye.Message = "Unexpected values ether."
		}
		errMsg = true
	}

	if errMsg {
		log.Printf("FireEye Status: %v. Message: %s Bitclout Deviation: %v. Ethereum Deviation: %v.\n", FireEye.Code, FireEye.Message, bitcloutDeviation, etherDeviation)
		log.Printf("Bitclout DB Balance: %v.  Bitclout Fees %v. Bitclout Wallet Balance: %v.\n", totalAccountBitclout, totalFeesBitclout, walletBitcloutBalance)
		log.Printf("Ethereum DB Balance: %v. Ethereum Fees: %v. Ethereum Wallet Balance %v.\n", totalAccountEther, totalFeesEther, walletEtherBalance)
	}

}

func SetSyncWarn(err error) {
	FireEye.Code = 10
	FireEye.Message = err.Error()
}

func checkErr(err error) {
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
}
