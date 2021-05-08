package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Pool struct {
	_id primitive.ObjectID `json: "_id" bson:"_id"`
	address string `json:"address" bson:"address"`
	active bool `json:"active" bson:"active"`
	listing primitive.ObjectID `json:"listing" bson:"listing"`
	balance float64 `json:"balance" bson:"balance"`
	privateKey PrivateKey `json:"privateKey" bson:"privateKey"`
	
}
type PrivateKey struct {
		salt string `json:"salt" bson:"salt"`
		encryptedKey string `json:"encryptedKey" bson:"encryptedKey"`
}