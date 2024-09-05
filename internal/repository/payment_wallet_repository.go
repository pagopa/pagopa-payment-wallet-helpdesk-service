package repository

import (
	"context"
	"fmt"
	"log"

	utils "pagopa.it/pagopa-payment-wallet-helpdesk-service/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const PaymentWalletWalletsCollectionNameEnvKey = "PAYMENT_WALLET_WALLETS_COLLECTION_NAME"

const PaymentWalletDBNameEnvKey = "MONGO_DB_NAME"

type paymentWalletRepository struct {
	collection *mongo.Collection
}

func NewPaymentWalletRepository(db *mongo.Client) *paymentWalletRepository {
	return &paymentWalletRepository{
		collection: db.Database(utils.GetEnvVariableOrDefault(PaymentWalletDBNameEnvKey, "wallet")).Collection(utils.GetEnvVariableOrDefault(PaymentWalletWalletsCollectionNameEnvKey, "payment-wallet")),
	}
}

func (p *paymentWalletRepository) GetWalletsByUserID(userID string, ctx context.Context) ([]WalletModel, error) {
	var wallets []WalletModel
	filter := bson.D{{Key: "userId", Value: userID}}
	log.Printf("Finding wallets for userId: [%s]", userID)
	cursor, err := p.collection.Find(ctx, filter)
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
