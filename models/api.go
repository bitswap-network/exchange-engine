package models

type CurrencyAmounts struct {
	Bitclout float64 `json:"totalBitclout" bson:"totalBitclout,omitempty" binding:"-"`
	Ether    float64 `json:"totalEther" bson:"totalEther,omitempty" binding:"-"`
}

type GetUsersStateLessResponse struct {
	Userlist []*UserList `json:"UserList"`
}

type UserList struct {
	BalanceNanos int64 `json:"BalanceNanos"`
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
