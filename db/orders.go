package db

import (
	"context"
	"errors"
	"log"
	"time"

	"exchange-engine/global"
	"exchange-engine/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetOrderFees(ctx context.Context) (*models.CurrencyAmounts, error) {
	var totalFees *models.CurrencyAmounts

	bitcloutMatchStage := bson.D{
		{"$match", bson.D{
			{"orderSide", "buy"},
		}},
	}
	bitcloutGroupStage := bson.D{
		{"$group", bson.D{
			{"_id", ""},
			{"totalBitclout", bson.D{
				{"$sum", "$fees"},
			}},
		}},
	}
	opts := options.Aggregate().SetMaxTime(2 * time.Second)
	cursor, err := OrderCollection().Aggregate(ctx, mongo.Pipeline{bitcloutMatchStage, bitcloutGroupStage}, opts)
	if err != nil {
		return nil, err
	}
	var resultsBclt []bson.M
	if err = cursor.All(ctx, &resultsBclt); err != nil {
		return nil, err
	}
	bsonBytes, _ := bson.Marshal(resultsBclt[0])
	if err = bson.Unmarshal(bsonBytes, &totalFees); err != nil {
		return nil, err
	}

	etherMatchStage := bson.D{
		{"$match", bson.D{
			{"orderSide", "sell"},
		}},
	}
	etherGroupStage := bson.D{
		{"$group", bson.D{
			{"_id", ""},
			{"totalEther", bson.D{
				{"$sum", "$fees"},
			}},
		}},
	}
	cursor, err = OrderCollection().Aggregate(ctx, mongo.Pipeline{etherMatchStage, etherGroupStage}, opts)
	if err != nil {
		return nil, err
	}
	var resultsEth []bson.M
	if err = cursor.All(ctx, &resultsEth); err != nil {
		return nil, err
	}
	bsonBytes, _ = bson.Marshal(resultsEth[0])
	if err = bson.Unmarshal(bsonBytes, &totalFees); err != nil {
		return nil, err
	}
	return totalFees, nil
}

func GetActiveOrders(ctx context.Context, publicKey string) (numOrders int, err error) {
	log.Printf("fetching num of active orders from : %s\n", publicKey)
	cursor, err := OrderCollection().Find(ctx, bson.M{"username": publicKey, "complete": false})
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		//Create a value into which the single document can be decoded
		var elem models.OrderSchema
		err := cursor.Decode(&elem)
		if err != nil {
			log.Panic(err)
		}
		numOrders += 1
	}
	return
}

func ValidateOrder(ctx context.Context, publicKey string, orderSide string, orderQuantity float64, totalEth float64) bool {
	log.Printf("fetching user balance from: %v\n", publicKey)
	// var userDoc *models.UserSchema
	userDoc, err := GetUserDoc(ctx, publicKey)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	if userDoc.Balance.InTransaction || orderQuantity > 500 || orderQuantity <= 0  {
		return false
	} else {
		if orderSide == "buy" {
			return totalEth <= userDoc.Balance.Ether
		} else {
			return orderQuantity <= userDoc.Balance.Bitclout
		}
	}
}

func CreateOrder(ctx context.Context, order *models.OrderSchema) error {
	log.Printf("create order: %v \n", order.OrderID)
	inTransaction, err := CheckUserTransactionState(ctx, order.Username)
	if err != nil {
		return err
	}
	numOrders, err := GetActiveOrders(ctx, order.Username)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if inTransaction {
		return errors.New("user in transaction")
	}
	if numOrders >= 10 {
		return errors.New("max active orders reached")
	} else {
		order.ID = primitive.NewObjectID()
		_, err := OrderCollection().InsertOne(ctx, order)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		log.Println("done creating order")
	}
	return nil
}

func CancelCompleteOrder(ctx context.Context, orderID string, errorString string) error {
	log.Printf("cancel complete: %v\n", orderID)

	update := bson.M{"$set": bson.M{"error": errorString, "complete": true, "completeTime": time.Now().UTC()}}
	_, err := OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

/*
Calculates the change in a user's bitclout and ether balances

Arguments:
	`ctx`: The context from which this function is called
	`orderSide`: Whether this is a BUY order or a sell order
	`quantity`: The quantity of BitClout/Ethereum bought/sold
	`totalPrice`: The total price previously sold
Returns:
	`bitcloutChange`: The change in the bitclout balance (BCLT)
	`etherChange`: The change in the ether balance ($)
	`fees`: The fees taken from the transaction ($)
*/
func calcChangeAndFees(ctx context.Context, orderSide string, quantity, totalPrice float64) (bitcloutChange, etherChange, fees float64) {
	ETHUSD := global.Exchange.ETHUSD
	if global.Exchange.ETHUSD == 0 {
		log.Printf("ETHUSD is 0. THIS IS NOT OK IF LIVE")
		ETHUSD = 2417.67 // value of eth usd as of June 16, 2021 11:04 PM ET
	}

	//update ether USD price var
	if orderSide == "buy" {
		fees = (quantity * global.Exchange.FEE)
		bitcloutChange = quantity - fees
		etherChange = -(totalPrice / ETHUSD)
	} else {
		fees = (totalPrice * global.Exchange.FEE) / ETHUSD
		bitcloutChange = -quantity
		etherChange = (totalPrice / ETHUSD) - fees
	}

	return bitcloutChange, etherChange, fees
}

func CompleteLimitOrder(ctx context.Context, orderID string, totalPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD

	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *models.OrderSchema

	//Find order in database
	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}

	bitcloutChange, etherChange, fees := calcChangeAndFees(ctx,
		orderDoc.OrderSide,
		orderDoc.OrderQuantity-orderDoc.OrderQuantityProcessed,
		totalPrice)

	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	execPrice := totalPrice / orderDoc.OrderQuantity
	// Mark the order as complete after bitclout and eth balances are modified
	// We can set `orderQuantityProcessed` since this order is completed.
	update := bson.M{"$set": bson.M{
		"orderQuantityProcessed": orderDoc.OrderQuantity,
		"complete":               true,
		"completeTime":           time.Now().UTC(),
		"execPrice":              execPrice,
	}, "$inc": bson.M{"fees": fees, "etherQuantity": (totalPrice / ETHUSD)}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}

	return nil
}

func CompleteLimitOrderDirect(ctx context.Context, orderID string) error {
	ETHUSD := global.Exchange.ETHUSD

	log.Printf("fulfill: %v\n", orderID)
	var orderDoc *models.OrderSchema

	//Finding order in database
	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		return err
	}

	bitcloutChange, etherChange, fees := calcChangeAndFees(ctx,
		orderDoc.OrderSide,
		orderDoc.OrderQuantity-orderDoc.OrderQuantityProcessed,
		((orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) * orderDoc.OrderPrice))

	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update := bson.M{"$set": bson.M{
		"orderQuantityProcessed": orderDoc.OrderQuantity,
		"complete":               true,
		"completeTime":           time.Now().UTC(),
		"execPrice":              orderDoc.OrderPrice,
	}, "$inc": bson.M{"fees": fees, "etherChange": (((orderDoc.OrderQuantity - orderDoc.OrderQuantityProcessed) * orderDoc.OrderPrice) / ETHUSD)}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}

	return nil
}

/*
Partially Complete a Limit Order
*/
func PartialLimitOrder(ctx context.Context, orderID string, quantityDelta float64, totalPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("partial fulfill: %v - %v\n", orderID, quantityDelta)
	var orderDoc *models.OrderSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err)
		return err
	}

	bitcloutChange, etherChange, fees := calcChangeAndFees(ctx,
		orderDoc.OrderSide,
		quantityDelta,
		totalPrice)

	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	execPrice := (orderDoc.ExecPrice*orderDoc.OrderQuantityProcessed + totalPrice) / (quantityDelta + orderDoc.OrderQuantityProcessed)
	// Mark the order as complete after bitclout and eth balances are modified
	update := bson.M{"$set": bson.M{"execPrice": execPrice},
		"$inc": bson.M{
			"fees":                   fees,
			"orderQuantityProcessed": quantityDelta,
			"etherQuantity":          (totalPrice / ETHUSD),
		},
	}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func PartialLimitOrderDirect(ctx context.Context, orderID string, quantityDelta float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("partial fulfill: %v - %v\n", orderID, quantityDelta)
	var orderDoc *models.OrderSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Println(err)
		return err
	}

	bitcloutChange, etherChange, fees := calcChangeAndFees(ctx,
		orderDoc.OrderSide,
		quantityDelta,
		(quantityDelta * orderDoc.OrderPrice))

	// attempt to modify bitclout balance and eth balance
	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	// INCREMENT the `orderQuantityProcessed` to reflect the partial quantity processed
	update := bson.M{"$set": bson.M{"execPrice": orderDoc.OrderPrice},
		"$inc": bson.M{
			"fees":                   fees,
			"orderQuantityProcessed": quantityDelta,
			"etherQuantity":          ((quantityDelta * orderDoc.OrderPrice) / ETHUSD),
		},
	}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}

func MarketOrder(ctx context.Context, orderID string, quantityProcessed float64, totalPrice float64) error {
	ETHUSD := global.Exchange.ETHUSD
	log.Printf("Fulfilling market order `%s` - Processed: %v\n", orderID, quantityProcessed)
	var orderDoc *models.OrderSchema

	err := OrderCollection().FindOne(ctx, bson.M{"orderID": orderID}).Decode(&orderDoc)
	if err != nil {
		log.Printf("Error fetching order `%s`: \n"+err.Error(), orderID)
		return err
	}
	bitcloutChange, etherChange, fees := calcChangeAndFees(ctx,
		orderDoc.OrderSide,
		quantityProcessed,
		totalPrice)

	log.Printf("bitChange: %v, etherChange: %v\n", bitcloutChange, etherChange)

	err = UpdateUserBalance(ctx, orderDoc.Username, bitcloutChange, etherChange)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	// Mark the order as complete after bitclout and eth balances are modified
	update := bson.M{"$set": bson.M{"etherQuantity": (totalPrice / ETHUSD), "fees": fees, "orderQuantityProcessed": quantityProcessed, "execPrice": (totalPrice / quantityProcessed), "complete": true, "completeTime": time.Now().UTC()}}
	_, err = OrderCollection().UpdateOne(ctx, bson.M{"orderID": orderID}, update)
	if err != nil {
		return err
	}
	return nil
}
