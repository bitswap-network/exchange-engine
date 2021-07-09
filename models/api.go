package models

type CurrencyAmounts struct {
	Bitclout uint64 `json:"totalBitclout" bson:"totalBitclout,omitempty" binding:"-"`
	Ether    uint64 `json:"totalEther" bson:"totalEther,omitempty" binding:"-"`
	Usdc     uint64 `json:"totalUsdc" bson:"totalUsdc,omitempty" binding:"-"`
}

type GetWalletBalanceBody struct {
	PublicKeyBase58Check string
	Confirmations        int64
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
