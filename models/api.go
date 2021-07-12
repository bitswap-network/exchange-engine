package models

import "math/big"

type CurrencyAmounts struct {
	Bitclout float64 `json:"totalBitclout" bson:"totalBitclout,omitempty" binding:"-"`
	Ether    float64 `json:"totalEther" bson:"totalEther,omitempty" binding:"-"`
}

type CurrencyAmountsBig struct {
	Bitclout *big.Int `json:"totalBitclout" bson:"totalBitclout,omitempty" binding:"-"`
	Ether    *big.Int `json:"totalEther" bson:"totalEther,omitempty" binding:"-"`
}

type TransferBalanceBody struct {
	SenderPrivateKeyBase58Check   string `json:"SenderPrivateKeyBase58Check"`
	RecipientPublicKeyBase58Check string `json:"RecipientPublicKeyBase58Check"`
	AmountNanos                   uint64 `json:"AmountNanos"`
	MinFeeRateNanosPerKB          int64  `json:"MinFeeRateNanosPerKB"`
	DryRun                        bool   `json:"DryRun"`
}

type TransferBalanceResponse struct {
	Error                string            `json:"Error"`
	Transaction          TBTransaction     `json:"Transaction"`
	TransactionInfo      TBTransactionInfo `json:"TransactionInfo"`
	AmountNanos          int64             `json:"AmountNanos"`
	MinFeeRateNanosPerKB int64             `json:"MinFeeRateNanosPerKB"`
	DryRun               bool              `json:"DryRun"`
}

type TBTransactionInfo struct {
	TotalInputNanos               uint64 `json:"TotalInputNanos"`
	SpendAmountNanos              uint64 `json:"SpendAmountNanos"`
	ChangeAmountNanos             uint64 `json:"ChangeAmountNanos"`
	FeeNanos                      uint64 `json:"FeeNanos"`
	FeeRateNanosPerKB             uint64 `json:"FeeRateNanosPerKB"`
	SenderPublicKeyBase58Check    string `json:"SenderPublicKeyBase58Check"`
	RecipientPublicKeyBase58Check string `json:"RecipientPublicKeyBase58Check"`
}

type TBTransaction struct {
	TransactionIDBase58Check string          `json:"TransactionIDBase58Check"`
	RawTransactionHex        string          `json:"RawTransactionHex"`
	Inputs                   []*TBTxnInputs  `json:"Inputs"`
	Outputs                  []*TBTxnOutputs `json:"Outputs"`
	SignatureHex             string          `json:"SignatureHex"`
	TransactionType          string          `json:"TransactionType"`
	BlockHashHex             string          `json:"BlockHashHex"`
}

type TBTxnInputs struct {
	TransactionIDBase58Check string `json:"AmountNanos"`
	Index                    int64  `json:"Index"`
}

type TBTxnOutputs struct {
	PublicKeyBase58Check string `json:"PublicKeyBase58Check"`
	AmountNanos          int64  `json:"AmountNanos"`
}

type GetWalletBalanceBody struct {
	PublicKeyBase58Check string `json:"PublicKeyBase58Check"`
	Confirmations        int64  `json:"Confirmations"`
}

type GetWalletBalanceResponse struct {
	ConfirmedBalanceNanos   uint64 `json:"ConfirmedBalanceNanos"`
	UnconfirmedBalanceNanos uint64 `json:"UnconfirmedBalanceNanos"`
}

type GetUsersStateLessResponse struct {
	Userlist []*UserList `json:"UserList"`
}

type UserList struct {
	BalanceNanos uint64 `json:"BalanceNanos"`
}
type CloutPriceAPI struct {
	Data float64 `json:"data"`
}

type EthPriceAPI struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Result  EthPriceAPIResult `json:"result"`
}
type EthPriceAPIResult struct {
	Ethbtc           string `json:"ethbtc"`
	Ethbtc_timestamp string `json:"ethbtc_timestamp"`
	Ethusd           string `json:"ethusd"`
	Ethusd_timestamp string `json:"ethusd_timestamp"`
}

type SanitizeRequest struct {
	PublicKey string `json:"publicKey" bson:"publicKey" binding:"required"`
}
