package fireeye

import (
	"context"
	"encoding/json"
	"log"
	"math"

	"exchange-engine/config"
	"exchange-engine/db"
	"exchange-engine/global"
	"exchange-engine/models"
)

type FireEyeT struct {
	Message     string
	Code int
}


//0-10 CODE -> OK, Possible info
//10-20 CODE -> Warn, Requests allowed still
//20-30 CODE -> Unavailable, blocking requests
//30-40 CODE -> Balance Error
var FireEye = &FireEyeT{
	Message: "Pending Initialization",
	Code:    20,
}
const OKTolerance = 0.0001 // 0.001% Tolerance
const MidTolerance = 0.001 // 0.1% Tolerance
const MaxTolerance = 0.01 //1% Tolerance

const initBcltTolerance = -156.90016130600003
const initEthTolerance = 24.346454580232383

func SyncStatus(ctx context.Context) {
	totalBalance, err := db.GetTotalBalances(ctx)
	if err != nil {
		log.Panic(err)
		return
	}
	totalFees, err := db.GetOrderFees(ctx)
	if err != nil {
		log.Panic(err)
		return
	}
	getUsersSLReqMap := map[string][]string{"PublicKeysBase58Check":config.Wallet.Addr_BCLT}
	getUsersSLReqBody, err := json.Marshal(getUsersSLReqMap)
	if err != nil {
		log.Panic(err)
		return
	}
	getUserSLResp := new(models.GetUsersStateLessResponse)
	if err := global.PostJson("https://bitclout.com/api/v0/get-users-stateless",getUsersSLReqBody,getUserSLResp); err != nil {
		log.Panic("ERROR getusersstateless: ", err)
		return
	}
	var walletBcltBalance float64 = 0
	for _,balance := range getUserSLResp.Userlist {
		walletBcltBalance += float64(balance.BalanceNanos)/1e9
	}
	pools, err := db.GetAllPools(ctx)
	if err != nil {
		log.Panic(err)
		return
	}
	var walletEthBalance float64 = 0
	for _,pool := range pools {
		walletEthBalance += pool.Balance
	}
	var (
		bitcloutSync = totalBalance.Bitclout + totalFees.Bitclout + initBcltTolerance
		etherSync = totalBalance.Ether + totalFees.Ether + initEthTolerance
	)

	if (math.Abs((bitcloutSync/walletBcltBalance)-1)>MaxTolerance) {
		FireEye.Code = 30
		FireEye.Message = "..."
	}else if (math.Abs((etherSync/walletEthBalance)-1)>MaxTolerance) {
		FireEye.Code = 31
		FireEye.Message = "..."
	} else {
		FireEye.Code = 0
		FireEye.Message = "OK"
	}
	log.Println(FireEye)
	return
}
