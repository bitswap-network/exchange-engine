package models

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type OrderSchema struct {
	ID bson.ObjectId `json:"_id" bson:"_id" binding:"-"`
	Username string `json:"username" bson:"username" binding:"required"`
	Created time.Time `json:"created" bson:"created" binding:"-"`
	OrderID string `json:"orderID" bson:"orderID" binding:"-"`
	OrderSide string `json:"orderSide" bson:"orderSide" binding:"required"`
	OrderType string `json:"orderType" bson:"orderType" binding:"required"`
	OrderQuantity float64 `json:"orderQuantity" bson:"orderQuantity" binding:"required"`
	OrderPrice float64 `json:"orderPrice" bson:"orderPrice" binding:"-"`
	OrderQuantityProcessed float64 `json:"orderQuantityProcessed" bson:"orderQuantityProcessed" binding:"-"`
	Complete bool `json:"complete" bson:"complete" binding:"-"`
	CompleteTime time.Time `json:"completeTime" bson:"completeTime" binding:"-"`
}

type UserSchema struct {
	ID bson.ObjectId `json:"_id" bson:"_id" binding:"-"`
	Username string `json:"username" bson:"username" binding:"required"`
	Email string `json:"email" bson:"email" binding:"-"`
	Password string `json:"password" bson:"password" binding:"-"`
	Balance UserBalance `json:"balance" bson:"balance" binding:"-"`
	OnGoingDeposit *bson.ObjectId `json:"onGoingDeposit" bson:"onGoingDeposit" binding:"-"`
	Transactions []*bson.ObjectId `json:"transactions" bson:"transactions" binding:"-"`
	Verification UserVerification `json:"verification" bson:"verification" binding:"-"`
	Bitclout UserBitclout `json:"bitclout" bson:"bitclout" binding:"-"`
	Created string `json:"created" bson:"created" binding:"-"`
	Admin bool `json:"admin" bson:"admin" binding:"-"`
}

type UserBalance struct {
	Bitclout float64 `json:"bitclout" bson:"bitclout" binding:"-"`
	Ether float64 `json:"ether" bson:"ether" binding:"-"`
}

type UserVerification struct {
	Email bool `json:"email" bson:"email" binding:"-"`
	EmailString string `json:"emailString" bson:"emailString" binding:"-"`
	PasswordString string `json:"passwordString" bson:"passwordString" binding:"-"`
	Status string `json:"status" bson:"status" binding:"-"`
	BitCloutString string `json:"bitcloutString" bson:"bitcloutString" binding:"-"`
}

type UserBitclout struct {
	PublicKey string `json:"publicKey" bson:"publicKey" binding:"-"`
	Bio *string `json:"bio" bson:"bio" binding:"-"`
	Verified bool `json:"verified" bson:"verified" binding:"-"`
	ProfilePicture *string `json:"profilePicture" bson:"profilePicture" binding:"-"`
}

