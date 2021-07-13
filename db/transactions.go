package db

import (
	"context"
	"errors"
	"exchange-engine/global"
	"exchange-engine/models"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreatePendingDeposit(ctx context.Context, user primitive.ObjectID, assetType string, value float64, txnHash string) error {
	var txnCheck *models.TransactionSchema
	err := TransactionCollection().FindOne(ctx, bson.M{"txnHash": txnHash}).Decode(&txnCheck)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in the collection
		if err == mongo.ErrNoDocuments {
			if assetType != "ETH" && assetType != "BCLT" && assetType != "USDC" {
				return errors.New("invalid asset type")
			}
			var txn models.TransactionSchema
			txn.ID = primitive.NewObjectID()
			txn.AssetType = assetType
			txn.Value = value
			txn.TransactionType = "deposit"
			txn.Completed = false
			txn.Created = time.Now().UTC()
			txn.User = user
			txn.State = "pending"
			txn.TxnHash = txnHash
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
		} else {
			return err
		}
	}

	return nil
}

func CompletePendingDeposit(ctx context.Context, user primitive.ObjectID, txnHash string, gasPrice uint64) error {
	update := bson.M{"$set": bson.M{"state": "done", "completed": true, "completionDate": time.Now().UTC(), "gasPrice": gasPrice}}
	_, err := TransactionCollection().UpdateOne(ctx, bson.M{"txnHash": txnHash}, update)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	log.Printf("completed txn: %v \n", txnHash)

	return nil
}
