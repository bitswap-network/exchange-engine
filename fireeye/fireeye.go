package fireeye

import (
	"context"
	"log"
	"math"

	"exchange-engine/config"
	"exchange-engine/db"
	"exchange-engine/global"
	"exchange-engine/models"
)

func SyncStatus(ctx context.Context) {

	TotalAccount, err := db.GetTotalBalances(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	totalAccountBitclout := global.FromNanos(TotalAccount.Bitclout)
	totalAccountEther := global.FromWei(TotalAccount.Ether)

	TotalFees, err := db.GetOrderFees(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}

	totalFeesBitclout := global.FromNanos(TotalFees.Bitclout)
	totalFeesEther := global.FromWei(TotalFees.Ether)

	walletBalance, err := GetMainWalletBalance(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	walletBitcloutBalanceNanos := walletBalance.ConfirmedBalanceNanos
	walletBitcloutBalance := global.FromNanos(walletBitcloutBalanceNanos)

	walletEtherBalanceWei, err := GetPoolsBalance(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	walletEtherBalance := global.FromWei(walletEtherBalanceWei)

	FireEye.WalletBalance = models.CurrencyAmounts{walletBitcloutBalanceNanos, walletEtherBalanceWei, 0}
	FireEye.TotalAccount = *TotalAccount
	FireEye.TotalFees = *TotalFees

	var (
		bitcloutSync = totalAccountBitclout + totalFeesBitclout + config.Wallet.InitBcltTolerance
		etherSync    = totalAccountEther + totalFeesEther + config.Wallet.InitEthTolerance
	)

	bitcloutDeviation := math.Abs((bitcloutSync / walletBitcloutBalance) - 1)
	etherDeviation := math.Abs((etherSync / walletEtherBalance) - 1)

	errMsg := false

	if bitcloutDeviation > MaxTolerance && etherDeviation > MaxTolerance {
		FireEye.Code = 33
		FireEye.Message = "Bitclout and ether balance out of sync (MAX TOLERANCE)."
		errMsg = true
	} else if bitcloutDeviation > MaxTolerance || etherDeviation > MaxTolerance {
		errMsg = true
		if bitcloutDeviation > MaxTolerance {
			FireEye.Code = 32
			FireEye.Message = "Bitclout balance out of sync (MAX TOLERANCE)."
		} else {
			FireEye.Code = 31
			FireEye.Message = "Ether balance out of sync (MAX TOLERANCE)."
		}
	} else if bitcloutDeviation > MidTolerance && etherDeviation > MidTolerance {
		FireEye.Code = 13
		FireEye.Message = "Bitclout and ether balance out of sync (MID TOLERANCE)."
		errMsg = true
	} else if bitcloutDeviation > MidTolerance || etherDeviation > MidTolerance {
		errMsg = true
		if bitcloutDeviation > MidTolerance {
			FireEye.Code = 12
			FireEye.Message = "Bitclout balance out of sync (MID TOLERANCE)."
		} else {
			FireEye.Code = 11
			FireEye.Message = "Ether balance out of sync (MID TOLERANCE)."

		}
	} else {
		FireEye.Code = 0
		FireEye.Message = "OK"
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
