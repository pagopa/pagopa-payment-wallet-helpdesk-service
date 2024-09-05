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

const MongoConnectionString = "mongodb://${MONGO_USERNAME}:${MONGO_PASSWORD}@${MONGO_HOSTNAME}:${MONGO_PORT}/${MONGO_DB_NAME}?ssl=${MONGO_SSL_ENABLED}"

func GetMongoClient() (*mongo.Client, error) {
	uri, err := utils.ReplaceEnvPropertiesInString(MongoConnectionString)
	if err != nil {
		errorMessage := fmt.Sprintf("Cannot instantiate Mongo client, error building connection string: [%s]", err)
		log.Println(errorMessage)
		return nil, errors.New(errorMessage)
	}
	additionalMongoConnectionProperties := utils.GetEnvVariableOrDefault("MONGO_ADDITIONAL_CONNECTION_PROPERTIES", "")
	mongoConnectionString := *uri + additionalMongoConnectionProperties
	client, err := mongo.Connect(context.Background(), options.Client().
		ApplyURI(mongoConnectionString))
	return client, err
}
