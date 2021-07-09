package db

import (
	"context"
	"exchange-engine/models"
	"log"

	"go.mongodb.org/mongo-driver/bson"
)

func GetAllWallets(ctx context.Context) ([]*models.WalletSchema, error) {
	var walletsArray []*models.WalletSchema

	cursor, err := WalletCollection().Find(ctx, bson.D{})
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		//Create a value into which the single document can be decoded
		var elem models.WalletSchema
		err := cursor.Decode(&elem)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		walletsArray = append(walletsArray, &elem)
	}
	return walletsArray, nil
}
func GetMainWallet(ctx context.Context) (*models.WalletSchema, error) {
	var walletDoc *models.WalletSchema

	err := WalletCollection().FindOne(ctx, bson.M{"super": 1}).Decode(&walletDoc)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return walletDoc, nil
}
