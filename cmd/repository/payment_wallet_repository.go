package repository

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	utils "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/utils"
)

const PaymentWalletWalletsCollectionNameEnvKey = "PAYMENT_WALLET_WALLETS_COLLECTION_NAME"

const PaymentWalletDBNameEnvKey = "MONGO_DB_NAME"

type PaymentWalletRepository struct {
	collection *mongo.Collection
}

func NewPaymentWalletRepository(db *mongo.Client) *PaymentWalletRepository {
	return &PaymentWalletRepository{
		collection: db.Database(utils.GetEnvVariableOrDefault(PaymentWalletDBNameEnvKey, "wallet")).Collection(utils.GetEnvVariableOrDefault(PaymentWalletWalletsCollectionNameEnvKey, "payment-wallets")),
	}
}

func (p *PaymentWalletRepository) GetWallets(userID string, walletStatus *string, walletType *string, ctx context.Context) ([]WalletModel, error) {
	var wallets []WalletModel
	filter := make(bson.D, 0)
	filter = append(filter, bson.E{Key: "userId", Value: userID})
	if walletStatus != nil {
		filter = append(filter, bson.E{Key: "status", Value: *walletStatus})
	}
	if walletType != nil {
		classDiscriminatorField := PaymentWalletDiscriminatorMap[*walletType]
		log.Printf("Wallet type: [%s], mapped to discriminator class value: [%s]", *walletType, classDiscriminatorField)
		filter = append(filter, bson.E{Key: "details._class", Value: classDiscriminatorField})
	}
	cursor, err := p.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error performing query searching for wallets: %v", err)
	}
	err = cursor.All(ctx, &wallets)
	if err != nil {
		return nil, fmt.Errorf("error while decoding retrived wallets: %v", err)
	}
	log.Printf("Found walles for filter: [%v] -> [%d]", filter, len(wallets))
	return wallets, nil
}
