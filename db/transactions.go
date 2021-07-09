package db

import (
	"context"
	"errors"
	"exchange-engine/global"
	"exchange-engine/models"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateDepositTransaction(ctx context.Context, user primitive.ObjectID, assetType string, value float64) error {
	if assetType != "ETH" && assetType != "BCLT" && assetType != "USDC" {
		return errors.New("invalid asset type")
	}
	var txn models.TransactionSchema
	txn.ID = primitive.NewObjectID()
	txn.AssetType = assetType
	txn.Value = value
	txn.TransactionType = "deposit"
	txn.Completed = true
	txn.Created = time.Now().UTC()
	txn.CompletionDate = time.Now().UTC()
	txn.Completed = true
	txn.User = user
	txn.State = "done"
	if assetType == "ETH" {
		txn.UsdValueAtTime = (global.Exchange.ETHUSD * value)
	} else if assetType == "BCLT" {
		txn.UsdValueAtTime = (global.Exchange.CLOUTUSD * value)
	} else {
		txn.UsdValueAtTime = value
	}
	log.Printf("create txn: %v \n", txn.ID)

	_, err := TransactionCollection().InsertOne(ctx, txn)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	log.Println("done creating txn")

	return nil
}
