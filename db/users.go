package db

import (
	"context"
	"errors"
	"log"
	"math"
	"time"

	"exchange-engine/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
Retrieves and populates a user doc from the database given the username

Arguments:
	ctx - The context from which the function is being called
	username - The username of the user you are searching for
	userDoc - The struct to hold the user document
*/
func GetUserDoc(ctx context.Context, publicKey string) (*models.UserSchema, error) {
	var userDoc *models.UserSchema
	err := UserCollection().FindOne(ctx, bson.M{"bitclout.publicKey": publicKey}).Decode(&userDoc)
	if err != nil {
		log.Printf("Could not find user: %v\n"+err.Error(), publicKey)
		return nil, err
	}
	return userDoc, nil
}

/*
Updates a user's BitClout and Ether balances by `bitcloutChange` and `etherChange` respectively.

One of `bitcloutChange` and `etherChange` MUST BE NEGATIVE. The other MUST BE POSITIVE.
*/
func UpdateUserBalance(ctx context.Context, publicKey string, bitcloutChange, etherChange float64) error {
	if (bitcloutChange > 0) == (etherChange > 0) {
		return errors.New("both `bitcloutChange` and `etherChange` cannot be positive or negative")
	}
	var (
		nanosChange = math.Round(bitcloutChange * 1e9)
		weiChange   = math.Round(etherChange * 1e18)
	)
	update := bson.M{"$inc": bson.M{"balance.bitclout": nanosChange, "balance.ether": weiChange}}
	_, err := UserCollection().UpdateOne(ctx, bson.M{"bitclout.publicKey": publicKey}, update)
	if err != nil {
		return err
	}
	return nil
}

func GetUserBalance(ctx context.Context, publicKey string) (balance *models.UserBalance, err error) {
	log.Printf("fetching user balance from: %v\n", publicKey)
	// var userDoc *models.UserSchema
	userDoc, err := GetUserDoc(ctx, publicKey)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	log.Println("done fetching balance")
	return userDoc.Balance, nil
}

func CheckUserTransactionState(ctx context.Context, publicKey string) (bool, error) {
	// var userDoc *models.UserSchema
	userDoc, err := GetUserDoc(ctx, publicKey)
	if err != nil {
		log.Println(err.Error())
		return false, err
	}
	return userDoc.Balance.InTransaction, nil
}

func GetUserOrders(ctx context.Context, publicKey string) ([]*models.OrderSchema, error) {
	var ordersArray []*models.OrderSchema
	cursor, err := OrderCollection().Find(ctx, bson.M{"username": publicKey, "complete": false})
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		//Create a value into which the single document can be decoded
		var elem models.OrderSchema
		err := cursor.Decode(&elem)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		ordersArray = append(ordersArray, &elem)
	}
	return ordersArray, nil
}

func GetTotalBalances(ctx context.Context) (*models.CurrencyAmounts, error) {
	var totalBalances *models.CurrencyAmounts

	balanceAggregateStage := bson.D{
		{"$group", bson.D{
			{"_id", ""},
			{"totalBitclout", bson.D{
				{"$sum", "$balance.bitclout"},
			}},
			{"totalEther", bson.D{
				{"$sum", "$balance.ether"},
			}},
			{"totalUsdc", bson.D{
				{"$sum", "$balance.usdc"},
			}},
		}},
	}
	opts := options.Aggregate().SetMaxTime(2 * time.Second)
	cursor, err := UserCollection().Aggregate(ctx, mongo.Pipeline{balanceAggregateStage}, opts)
	if err != nil {
		return nil, err
	}
	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	bsonBytes, _ := bson.Marshal(results[0])
	err = bson.Unmarshal(bsonBytes, &totalBalances)
	if err != nil {
		return nil, err
	}

	return totalBalances, nil
}
