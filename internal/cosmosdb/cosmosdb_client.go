package cosmosdb

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	utils "pagopa.it/pagopa-payment-wallet-helpdesk-service/internal/utils"
)

const MongoConnectionString = "mongodb://${MONGO_USERNAME}:${MONGO_PASSWORD}@${MONGO_HOSTNAME}:${MONGO_PORT}/?ssl=${MONGO_SSL_ENABLED}"

var client *mongo.Client

func GetMongoClient() (*mongo.Client, error) {
	if client != nil {
		return client, nil
	}
	uri, err := utils.ReplaceEnvPropertiesInString(MongoConnectionString)
	if err != nil {
		errorMessage := fmt.Sprintf("error: Cannot instantiate Mongo client, error building connection string: [%s]", err)
		log.Println(errorMessage)
		return nil, errors.New(errorMessage)
	}
	additionalMongoConnectionProperties := utils.GetEnvVariableOrDefault("MONGO_ADDITIONAL_CONNECTION_PROPERTIES", "")
	mongoConnectionString := *uri + additionalMongoConnectionProperties
	mongoClient, err := mongo.Connect(context.Background(), options.Client().
		ApplyURI(mongoConnectionString).SetDirect(true))
	client = mongoClient
	return client, err
}

func CloseMongoClient() {
	if client != nil {
		err := client.Disconnect(context.Background())
		if err != nil {
			log.Fatalf("Error closing mongo client: %v", err)
		}
		log.Println("Mongo client disconnected successfully")
	}
}
