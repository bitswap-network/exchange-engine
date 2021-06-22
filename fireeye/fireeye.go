package fireeye

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"

	"exchange-engine/config"
	"exchange-engine/db"
	"exchange-engine/global"
	"exchange-engine/models"
)

type FireEyeT struct {
	Message string
	Code    int
}

//0-10 CODE -> OK, Possible info
//10-20 CODE -> Warn, Requests allowed still
//20-30 CODE -> Unavailable, blocking requests
//30-40 CODE -> Balance Error
var FireEye = &FireEyeT{
	Message: "Pending Initialization",
	Code:    20,
}

const MidTolerance = 0.001 // 0.1% Tolerance
const MaxTolerance = 0.005 //0.5% Tolerance

const initBcltTolerance = -159.230163
const initEthTolerance = 10.8421526

func SyncStatus(ctx context.Context) {

	totalBalance, err := db.GetTotalBalances(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	totalFees, err := db.GetOrderFees(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	getUsersSLReqMap := map[string][]string{"PublicKeysBase58Check": config.Wallet.Addr_BCLT}
	getUsersSLReqBody, err := json.Marshal(getUsersSLReqMap)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	getUserSLResp := new(models.GetUsersStateLessResponse)
	if err := global.PostJson("https://bitclout.com/api/v0/get-users-stateless", getUsersSLReqBody, getUserSLResp); err != nil {
		SetSyncWarn(err)
		log.Panic("ERROR getusersstateless: ", err)
		return
	}
	// We should only retrieve our single BitClout wallet account
	if len(getUserSLResp.Userlist) != len(config.Wallet.Addr_BCLT) {
		SetSyncWarn(errors.New("could not find the Wallet BitClout Account"))
		log.Panic("ERROR getusersstateless UserList too small")
	}
	var walletBcltBalance float64 = 0
	for _, balance := range getUserSLResp.Userlist {
		walletBcltBalance += float64(balance.BalanceNanos) / 1e9
	}
	pools, err := db.GetAllPools(ctx)
	if err != nil {
		SetSyncWarn(err)
		log.Panic(err)
		return
	}
	var walletEthBalance float64 = 0
	for _, pool := range pools {
		walletEthBalance += pool.Balance
	}
	var (
		bitcloutSync = totalBalance.Bitclout + totalFees.Bitclout + config.Wallet.InitBcltTolerance
		etherSync    = totalBalance.Ether + totalFees.Ether + config.Wallet.InitEthTolerance
	)
	bitcloutDeviation := math.Abs((bitcloutSync / walletBcltBalance) - 1)
	etherDeviation := math.Abs((etherSync / walletEthBalance) - 1)

	if bitcloutDeviation > MaxTolerance && etherDeviation > MaxTolerance {
		FireEye.Code = 33
		FireEye.Message = "Bitclout and ether balance out of sync (MAX TOLERANCE)."
		log.Printf("FireEye Status: %v. Message: %s. Bitclout Deviation: %v. Ethereum Deviation: %v. Bitclout DB Balance: %v. Bitclout Wallet Balance: %v. Ethereum DB Balance: %v. Ethereum Wallet Balance %v.\n", FireEye.Code, FireEye.Message, bitcloutDeviation, etherDeviation, bitcloutSync-config.Wallet.InitBcltTolerance, walletBcltBalance, etherSync-config.Wallet.InitEthTolerance, walletEthBalance)
	} else if bitcloutDeviation > MaxTolerance || etherDeviation > MaxTolerance {
		if bitcloutDeviation > MaxTolerance {
			FireEye.Code = 32
			FireEye.Message = "Bitclout balance out of sync (MAX TOLERANCE)."
			log.Printf("FireEye Status: %v. Message: %s. Bitclout Deviation: %v. Ethereum Deviation: %v. Bitclout DB Balance: %v. Bitclout Wallet Balance: %v. Ethereum DB Balance: %v. Ethereum Wallet Balance %v.\n", FireEye.Code, FireEye.Message, bitcloutDeviation, etherDeviation, bitcloutSync-config.Wallet.InitBcltTolerance, walletBcltBalance, etherSync-config.Wallet.InitEthTolerance, walletEthBalance)
		} else {
			FireEye.Code = 31
			FireEye.Message = "Ether balance out of sync (MAX TOLERANCE)."
			log.Printf("FireEye Status: %v. Message: %s. Bitclout Deviation: %v. Ethereum Deviation: %v. Bitclout DB Balance: %v. Bitclout Wallet Balance: %v. Ethereum DB Balance: %v. Ethereum Wallet Balance %v.\n", FireEye.Code, FireEye.Message, bitcloutDeviation, etherDeviation, bitcloutSync-config.Wallet.InitBcltTolerance, walletBcltBalance, etherSync-config.Wallet.InitEthTolerance, walletEthBalance)
		}
	} else if bitcloutDeviation > MidTolerance && etherDeviation > MidTolerance {
		FireEye.Code = 13
		FireEye.Message = "Bitclout and ether balance out of sync (MID TOLERANCE)."
		log.Printf("FireEye Status: %v. Message: %s. Bitclout Deviation: %v. Ethereum Deviation: %v. Bitclout DB Balance: %v. Bitclout Wallet Balance: %v. Ethereum DB Balance: %v. Ethereum Wallet Balance %v.\n", FireEye.Code, FireEye.Message, bitcloutDeviation, etherDeviation, bitcloutSync-config.Wallet.InitBcltTolerance, walletBcltBalance, etherSync-config.Wallet.InitEthTolerance, walletEthBalance)
	} else if bitcloutDeviation > MidTolerance || etherDeviation > MidTolerance {
		if bitcloutDeviation > MidTolerance {
			FireEye.Code = 12
			FireEye.Message = "Bitclout balance out of sync (MID TOLERANCE)."
			log.Printf("FireEye Status: %v. Message: %s. Bitclout Deviation: %v. Ethereum Deviation: %v. Bitclout DB Balance: %v. Bitclout Wallet Balance: %v. Ethereum DB Balance: %v. Ethereum Wallet Balance %v.\n", FireEye.Code, FireEye.Message, bitcloutDeviation, etherDeviation, bitcloutSync-config.Wallet.InitBcltTolerance, walletBcltBalance, etherSync-config.Wallet.InitEthTolerance, walletEthBalance)
		} else {
			FireEye.Code = 11
			FireEye.Message = "Ether balance out of sync (MID TOLERANCE)."
			log.Printf("FireEye Status: %v. Message: %s. Bitclout Deviation: %v. Ethereum Deviation: %v. Bitclout DB Balance: %v. Bitclout Wallet Balance: %v. Ethereum DB Balance: %v. Ethereum Wallet Balance %v.\n", FireEye.Code, FireEye.Message, bitcloutDeviation, etherDeviation, bitcloutSync-config.Wallet.InitBcltTolerance, walletBcltBalance, etherSync-config.Wallet.InitEthTolerance, walletEthBalance)
		}
	} else {
		FireEye.Code = 0
		FireEye.Message = "OK"
	}

}

func SetSyncWarn(err error) {
	FireEye.Code = 10
	FireEye.Message = err.Error()
}
