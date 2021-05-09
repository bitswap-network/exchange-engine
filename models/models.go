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
	OrderQuantity float64 `json:"orderQuantity" bson:"orderQuantity" binding:"required"`
	OrderPrice float64 `json:"orderPrice" bson:"orderPrice" binding:"-"`
	PartialQuantityProcessed float64 `json:"partialQuantityProcessed" bson:"partialQuantityProcessed" binding:"-"`
	Complete bool `json:"complete" bson:"complete" binding:"-"`
	CompleteTime time.Time `json:"completeTime" bson:"completeTime" binding:"-"`
}