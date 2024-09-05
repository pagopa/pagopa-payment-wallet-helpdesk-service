package repository

import (
	"context"
	"fmt"
	"log"
	"sync"

	C "pagopa.it/pagopa-payment-wallet-helpdesk-service/internal/cosmosdb"
	utils "pagopa.it/pagopa-payment-wallet-helpdesk-service/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	once                    sync.Once
	mongoClient             *mongo.Client
	paymentWalletCollection *mongo.Collection
	err                     error
)

const PaymentWalletWalletsCollectionNameEnvKey = "PAYMENT_WALLET_WALLETS_COLLECTION_NAME"

const PaymentWalletDBNameEnvKey = "MONGO_DB_NAME"

// initialize this repository instantiating Mongo client and retrieving payment wallet collection
func init() {
	once.Do(
		func() {
			log.Println("Initializing Mongo client")
			mongoClient, err = C.GetMongoClient(context.Background())
			if err != nil {
				log.Fatalf("Error connecting to Mongo DB: %v", err)
			}
			dbName := utils.GetEnvVariableOrDefault(PaymentWalletDBNameEnvKey, "wallet")
			paymentWalletCollectioName := utils.GetEnvVariableOrDefault(PaymentWalletWalletsCollectionNameEnvKey, "wallet")
			log.Printf("Payment wallet DB name: [%s], wallets collection: [%s]", dbName, paymentWalletCollectioName)
			if utils.AnyEmpty(dbName, paymentWalletCollectioName) {
				log.Fatalf("Payment wallet db name or payment wallets collection name null not valid, check for %s system properties to be properly set", [2]string{PaymentWalletDBNameEnvKey, PaymentWalletWalletsCollectionNameEnvKey})
			}
			paymentWalletCollection = mongoClient.Database(dbName).Collection(paymentWalletCollectioName)
			log.Printf("Payment wallet retrieved collection: %s", paymentWalletCollectioName)
		})
}

func GetWalletsByUserID(userID string, ctx context.Context) ([]WalletModel, error) {
	var wallets []WalletModel
	filter := bson.D{{Key: "userId", Value: userID}}
	log.Printf("Finding wallets for userId: [%s]", userID)
	cursor, err := paymentWalletCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error performing query searching for wallets: %v", err)
	}
	err = cursor.All(ctx, &wallets)
	if err != nil {
		return nil, fmt.Errorf("error while decoding retrived wallets: %v", err)
	}
	log.Printf("Found walles for userId: [%s] -> [%d]", userID, len(wallets))
	return wallets, nil
}
