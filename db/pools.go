package db

import (
	"context"
	"exchange-engine/models"
	"log"

	"go.mongodb.org/mongo-driver/bson"
)



func GetAllPools(ctx context.Context) ([]*models.PoolSchema, error) {
	log.Printf("fetching pool balances: \n")
	var poolsArray []*models.PoolSchema
	
	cursor, err := PoolCollections().Find(ctx,bson.D{})
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		//Create a value into which the single document can be decoded
		var elem models.PoolSchema
		err := cursor.Decode(&elem)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		poolsArray = append(poolsArray, &elem)
	}
	return poolsArray, nil
}