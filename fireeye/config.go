package fireeye

import "exchange-engine/models"

type FireEyeT struct {
	Message       string
	Code          int
	TotalAccount  models.CurrencyAmountsBig
	TotalFees     models.CurrencyAmountsBig
	WalletBalance models.CurrencyAmounts
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
const BitcloutConfirmations = 0
const EthereumConfirmations = 8
