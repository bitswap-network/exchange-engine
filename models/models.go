package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderSchema struct {
	ID primitive.ObjectID `json: "_id" bson:"_id" binding:"-"`
	Username string `json: "username" bson: "username" binding:"required"`
	Created time.Time `json:"created" bson:"created" binding:"-"`
	OrderID string `json: "orderID" bson:"orderID" binding:"-"`
	OrderSide string `json:"orderSide" bson:"orderSide" binding:"required"`
	OrderQuantity float32 `json:"orderQuantity" bson:"orderQuantity" binding:"required"`
	OrderPrice float32 `json:"orderPrice" bson:"orderPrice" binding:"-"`
	PartialQuantityProcessed float32 `json:"partialQuantityProcessed" bson:"partialQuantityProcessed" binding:"-"`
	QuantityLeft float32 `json:"quantityLeft" bson:"quantityLeft" binding:"-"`
}