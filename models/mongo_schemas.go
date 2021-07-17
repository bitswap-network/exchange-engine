package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransactionSchema struct {
	ID              primitive.ObjectID `json:"_id" bson:"_id,omitempty" binding:"-"`
	User            primitive.ObjectID `json:"user" bson:"user" binding:"required"`
	TransactionType string             `json:"transactionType" bson:"transactionType" binding:"required"`
	AssetType       string             `json:"assetType" bson:"assetType" binding:"required"`
	Value           float64            `json:"value" bson:"value" binding:"required"`
	UsdValueAtTime  float64            `json:"usdValueAtTime" bson:"usdValueAtTime" binding:"-"`
	Created         time.Time          `json:"created" bson:"created" binding:"required"`
	Completed       bool               `json:"completed" bson:"completed" binding:"required"`
	CompletionDate  time.Time          `json:"completionDate" bson:"completionDate,omitempty" binding:"-"`
	State           string             `json:"state" bson:"state,omitempty" binding:"-"`
	Error           string             `json:"error" bson:"error,omitempty" binding:"-"`
	PoolAddress     *string            `json:"poolAddress" bson:"poolAddress,omitempty" binding:"-"`
	GasPrice        *float64           `json:"gasPrice" bson:"gasPrice,omitempty" binding:"-"`
	TxnHash         string             `json:"txnHash" bson:"txnHash,omitempty" binding:"-"`
}

type PoolSchema struct {
	ID          primitive.ObjectID  `json:"_id" bson:"_id,omitempty" binding:"-"`
	Active      bool                `json:"active" bson:"active" binding:"required"`
	Super       int                 `json:"super" bson:"super" binding:"required"`
	Balance     PoolBalanceSchema   `json:"balance" bson:"balance" binding:"required"`
	Address     string              `json:"address" bson:"address" binding:"required"`
	HashedKey   string              `json:"hashedKey" bson:"hashedKey" binding:"required"`
	ActiveStart *uint64             `json:"activeStart" bson:"activeStart" binding:"-"`
	User        *primitive.ObjectID `json:"user" bson:"user,omitempty" binding:"-"`
	TxnHashList []string            `json:"txnHashList" bson:"txnHashList,omitempty" binding:"-"`
}

type PoolBalanceSchema struct {
	ETH  float64 `json:"eth" bson:"eth" binding:"required"`
	USDC float64 `json:"usdc" bson:"usdc" binding:"required"`
}

type WalletSchema struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id,omitempty" binding:"required"`
	KeyInfo KeyInfoSchema      `json:"keyInfo" bson:"keyInfo" binding:"required"`
	User    primitive.ObjectID `json:"user" bson:"user,omitempty" binding:"-"`
	Fees    FeesSchema         `json:"balance" bson:"balance" binding:"required"`
	Super   uint               `json:"super" bson:"super" binding:"required"`
	Status  uint               `json:"status" bson:"status" binding:"required"`
}

type FeesSchema struct {
	Bitclout uint64 `json:"bitclout" bson:"bitclout" binding:"required"`
}

type KeyInfoSchema struct {
	Bitclout BitcloutKeyInfo `json:"bitclout" bson:"bitclout" binding:"required"`
}

type BitcloutKeyInfo struct {
	PublicKeyBase58Check  string `json:"publicKeyBase58Check" bson:"publicKeyBase58Check" binding:"required"`
	PublicKeyHex          string `json:"publicKeyHex" bson:"publicKeyHex" binding:"required"`
	PrivateKeyBase58Check string `json:"privateKeyBase58Check" bson:"privateKeyBase58Check" binding:"required"`
	PrivateKeyHex         string `json:"privateKeyHex" bson:"privateKeyHex" binding:"required"`
	ExtraText             string `json:"extraText" bson:"extraText" binding:"required"`
	Index                 uint64 `json:"index" bson:"index" binding:"required"`
}

type DepthSchema struct {
	ID        primitive.ObjectID  `json:"_id" bson:"_id,omitempty" binding:"-"`
	TimeStamp time.Time           `json:"timestamp" bson:"timestamp" binding:"-"`
	Asks      []*PriceLevelSchema `json:"asks" bson:"asks" binding:"-"`
	Bids      []*PriceLevelSchema `json:"bids" bson:"bids" binding:"-"`
}
type PriceLevelSchema struct {
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
	Quote                  float64            `json:"quote" binding:"-"`
	Tolerance              float64            `json:"tolerance" binding:"-"`
}

type UserSchema struct {
	ID           primitive.ObjectID `json:"_id" bson:"_id" binding:"-"`
	Name         string             `json:"name" bson:"name" binding:"-"`
	Email        string             `json:"email" bson:"email" binding:"-"`
	Password     string             `json:"password" bson:"password" binding:"-"`
	Balance      *UserBalance       `json:"balance" bson:"balance" binding:"-"`
	Verification UserVerification   `json:"verification" bson:"verification" binding:"-"`
	Bitclout     UserBitclout       `json:"bitclout" bson:"bitclout" binding:"-"`
	Tier         uint               `json:"tier" bson:"tier" binding:"required"`
	Created      time.Time          `json:"created" bson:"created" binding:"-"`
	Admin        bool               `json:"admin" bson:"admin" binding:"-"`
}

type UserBalance struct {
	Bitclout      uint64 `json:"bitclout" bson:"bitclout" binding:"required"`
	Ether         uint64 `json:"ether" bson:"ether" binding:"required"`
	USDC          uint64 `json:"usdc" bson:"usdc" binding:"required"`
	InTransaction bool   `json:"in_transaction" bson:"in_transaction" binding:"required"`
}

type UserVerification struct {
	Email            bool   `json:"email" bson:"email" binding:"-"`
	EmailString      string `json:"emailString" bson:"emailString" binding:"-"`
	PersonaAccountId string `json:"personaAccountId" bson:"personaAccountId" binding:"-"`
	InquiryId        string `json:"inquiryId" bson:"inquiryId" binding:"-"`
	PersonaVerified  bool   `json:"personaVerified" bson:"personaVerified" binding:"required"`
}

type UserBitclout struct {
	PublicKey string  `json:"publicKey" bson:"publicKey" binding:"required"`
	Bio       *string `json:"bio" bson:"bio" binding:"-"`
	Verified  bool    `json:"verified" bson:"verified" binding:"-"`
	Username  string  `json:"username" bson:"username" binding:"-"`
}
