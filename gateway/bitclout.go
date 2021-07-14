package gateway

import (
	"context"
	"encoding/json"
	"exchange-engine/config"
	"exchange-engine/db"
	"exchange-engine/fireeye"
	"exchange-engine/global"
	"exchange-engine/models"
	"fmt"
	"log"
	"sync"
	"time"
)

func QueryWallets(ctx context.Context) {
	start := time.Now()
	defer func() {
		log.Println("Execution Time: ", time.Since(start))
	}()
	wg := sync.WaitGroup{}
	wallets, err := db.GetAllWallets(ctx)
	if err != nil {
		log.Panic(err)
		return
	}
	for _, wallet := range wallets {
		wg.Add(1)
		time.Sleep(20 * time.Millisecond) // to prevent api from getting overwhelmed
		go func(wallet *models.WalletSchema) {
			if !config.IsTest {
				walletBalance, txns, err := GetWalletBalance(wallet)
				if err != nil {
					log.Panic(err)
				}
				// log.Println(walletBalance, wallet.KeyInfo.Bitclout.PublicKeyBase58Check)
				for _, txn := range txns {
					if txn.AmountNanos-wallet.Fees.Bitclout > 100000 && txn.PublicKeyBase58Check == wallet.KeyInfo.Bitclout.PublicKeyBase58Check {
						log.Println("found deposit: ", txn)
						if txn.Confirmations == 0 {
							amountToTransfer := txn.AmountNanos - BITCLOUT_DEPOSIT_FEENANOS
							err = db.CreatePendingDeposit(ctx, wallet.User, "BCLT", global.FromNanos(amountToTransfer), txn.TransactionIDBase58Check)
							if err != nil {
								log.Panic(err)
							}
						} else {
							if walletBalance-wallet.Fees.Bitclout >= txn.AmountNanos {
								log.Println("completing deposit")

								err = db.CompletePendingDeposit(ctx, wallet.User, txn.TransactionIDBase58Check, BITCLOUT_DEPOSIT_FEENANOS)
								if err != nil {
									log.Println(err)
								} else {
									amountToTransfer := txn.AmountNanos - BITCLOUT_DEPOSIT_FEENANOS
									transaction, err := TransferToMain(ctx, wallet, amountToTransfer, false)
									if err != nil {
										log.Panic(err)
									}
									log.Println(transaction)
									feesRemaining := BITCLOUT_DEPOSIT_FEENANOS - transaction.TransactionInfo.FeeNanos

									err = db.CreditUserBalance(ctx, wallet.User, amountToTransfer, 0)
									if err != nil {
										log.Println(err)
									}

									err = db.SetFeesBitclout(ctx, wallet, feesRemaining)
									if err != nil {
										log.Println(err)
									}
								}
							}
						}
						//create transaction in database
						//send funds
						//update user balance
					}
				}
			}
			wg.Done()
		}(wallet)
	}
	wg.Wait()
}

func TransferToMain(ctx context.Context, wallet *models.WalletSchema, amountNanos uint64, dryRun bool) (transferBalanceResponse *models.TransferBalanceResponse, err error) {
	log.Println("transfering to main")
	mainWallet, err := db.GetMainWallet(ctx)
	if err != nil {
		log.Println(err)
		return
	}
	senderPrivateKey, err := global.DecryptGCM(wallet.KeyInfo.Bitclout.PrivateKeyBase58Check, config.Wallet.HashKey)
	if err != nil {
		log.Println(err)
		return
	}
	transferBalanceMap := models.TransferBalanceBody{senderPrivateKey, mainWallet.KeyInfo.Bitclout.PublicKeyBase58Check, amountNanos, MinFeeRateNanosPerKB, dryRun}
	transferBalanceReqBody, err := json.Marshal(transferBalanceMap)
	if err != nil {
		log.Println(err)
		return
	}
	transferBalanceResponse = new(models.TransferBalanceResponse)
	err = global.PostJson(fmt.Sprintf("%s/api/v1/transfer-bitclout", config.BITCLOUT_NODEURL), transferBalanceReqBody, transferBalanceResponse)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func GetWalletBalance(wallet *models.WalletSchema) (confirmedBalance uint64, transactions []*models.UTXOResp, err error) {
	getWalletBalanceMap := models.GetWalletBalanceBody{wallet.KeyInfo.Bitclout.PublicKeyBase58Check, fireeye.BitcloutConfirmations}
	getWalletBalanceReqBody, _ := json.Marshal(getWalletBalanceMap)
	if err != nil {
		log.Println(err)
	}
	getWalletBalanceResp := new(models.GetWalletBalanceResponse)
	err = global.PostJson(fmt.Sprintf("%s/api/v1/balance", config.BITCLOUT_NODEURL), getWalletBalanceReqBody, getWalletBalanceResp)

	confirmedBalance = getWalletBalanceResp.ConfirmedBalanceNanos + getWalletBalanceResp.UnconfirmedBalanceNanos
	transactions = getWalletBalanceResp.UTXOs
	return
}

// func CreateDeposit(ctx context.Context,wallet *models.WalletSchema) (balance uint64, err error) {
// 	err := db.CreateDepositTransaction(ctx,wallet.User,"BCLT")
// 	return
// }
