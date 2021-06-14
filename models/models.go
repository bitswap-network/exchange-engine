package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CurrencyAmounts struct {
	Bitclout float64 `json:"totalBitclout" bson:"totalBitclout,omitempty" binding:"-"`
	Ether    float64 `json:"totalEther" bson:"totalEther,omitempty" binding:"-"`
}

type PoolSchema struct {
	ID          primitive.ObjectID  `json:"_id" bson:"_id,omitempty" binding:"-"`
	Active      bool                `json:"active" bson:"active" binding:"required"`
	Super       int                 `json:"super" bson:"super" binding:"required"`
	Balance     float64             `json:"balance" bson:"balance" binding:"required"`
	Address     string              `json:"address" bson:"address" binding:"required"`
	HashedKey   string              `json:"hashedKey" bson:"hashedKey" binding:"required"`
	ActiveStart *int64              `json:"activeStart" bson:"activeStart" binding:"-"`
	User        *primitive.ObjectID `json:"user" bson:"user,omitempty" binding:"-"`
	TxnHashList []string            `json:"txnHashList" bson:"txnHashList,omitempty" binding:"-"`
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

type UsernameRequest struct {
	Username string `json:"username" bson:"username" binding:"required"`
}

type DepthSchema struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty" binding:"-"`
	TimeStamp time.Time          `json:"timestamp" bson:"timestamp" binding:"-"`
	Asks      []*PriceLevel      `json:"asks" bson:"asks" binding:"-"`
	Bids      []*PriceLevel      `json:"bids" bson:"bids" binding:"-"`
}
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}
type OrderSchema struct {
	ID                     primitive.ObjectID `json:"_id" bson:"_id,omitempty" binding:"-"`
	Username               string             `json:"username" bson:"username" binding:"required"`
	Created                time.Time          `json:"created" bson:"created,omitempty" binding:"-"`
	OrderID                string             `json:"orderID" bson:"orderID" binding:"-"`
	OrderSide              string             `json:"orderSide" bson:"orderSide" binding:"required"`
	OrderType              string             `json:"orderType" bson:"orderType" binding:"-"`
	Fees                   float64            `json:"fees" bson:"fees" binding:"-"`
	EtherQuantity          float64            `json:"etherQuantity" bson:"etherQuantity" binding:"-"`
	OrderQuantity          float64            `json:"orderQuantity" bson:"orderQuantity" binding:"required"`
	OrderPrice             float64            `json:"orderPrice,omitempty" bson:"orderPrice,omitempty" binding:"-"`
	ExecPrice              float64            `json:"execPrice,omitempty" bson:"execPrice,omitempty" binding:"-"`
	OrderQuantityProcessed float64            `json:"orderQuantityProcessed" bson:"orderQuantityProcessed" binding:"-"`
	Complete               bool               `json:"complete" bson:"complete" binding:"-"`
	Error                  string             `json:"error" bson:"error" binding:"-"`
	CompleteTime           time.Time          `json:"completeTime" bson:"completeTime,omitempty" binding:"-"`
}

type UserSchema struct {
	ID             primitive.ObjectID    `json:"_id" bson:"_id" binding:"-"`
	Username       string                `json:"username" bson:"username" binding:"required"`
	Email          string                `json:"email" bson:"email" binding:"-"`
	Password       string                `json:"password" bson:"password" binding:"-"`
	Balance        *UserBalance          `json:"balance" bson:"balance" binding:"-"`
	OnGoingDeposit *primitive.ObjectID   `json:"onGoingDeposit" bson:"onGoingDeposit" binding:"-"`
	Transactions   []*primitive.ObjectID `json:"transactions" bson:"transactions" binding:"-"`
	Verification   UserVerification      `json:"verification" bson:"verification" binding:"-"`
	Bitclout       UserBitclout          `json:"bitclout" bson:"bitclout" binding:"-"`
	Created        time.Time             `json:"created" bson:"created" binding:"-"`
	Admin          bool                  `json:"admin" bson:"admin" binding:"-"`
}

type UserBalance struct {
	Bitclout      float64 `json:"bitclout" bson:"bitclout" binding:"-"`
	Ether         float64 `json:"ether" bson:"ether" binding:"-"`
	InTransaction bool    `json:"in_transaction" bson:"in_transaction" binding:"-"`
}

type UserVerification struct {
	Email          bool   `json:"email" bson:"email" binding:"-"`
	EmailString    string `json:"emailString" bson:"emailString" binding:"-"`
	PasswordString string `json:"passwordString" bson:"passwordString" binding:"-"`
	Status         string `json:"status" bson:"status" binding:"-"`
	BitCloutString string `json:"bitcloutString" bson:"bitcloutString" binding:"-"`
}

type UserBitclout struct {
	PublicKey      string  `json:"publicKey" bson:"publicKey" binding:"-"`
	Bio            *string `json:"bio" bson:"bio" binding:"-"`
	Verified       bool    `json:"verified" bson:"verified" binding:"-"`
	ProfilePicture *string `json:"profilePicture" bson:"profilePicture" binding:"-"`
}
