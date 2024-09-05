package cosmosdb

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestGetMongoClientShouldReturnClientSuccessfully(t *testing.T) {
	os.Setenv("MONGO_USERNAME", "username")
	os.Setenv("MONGO_PASSWORD", "password")
	os.Setenv("MONGO_HOSTNAME", "localhost")
	os.Setenv("MONGO_PORT", "1234")
	os.Setenv("MONGO_DB_NAME", "dbName")
	os.Setenv("MONGO_SSL_ENABLED", "false")
	client, err := GetMongoClient(context.Background())
	if err != nil {
		t.FailNow()
	}
	if client == nil {
		t.FailNow()
	}
}

func TestGetMongoClientShouldReturnErrorForMissingSystemPropertyValue(t *testing.T) {
	os.Setenv("MONGO_USERNAME", "username")
	os.Setenv("MONGO_PASSWORD", "password")
	os.Setenv("MONGO_HOSTNAME", "localhost")
	os.Setenv("MONGO_PORT", "1234")
	os.Setenv("MONGO_DB_NAME", "dbName")
	os.Setenv("MONGO_SSL_ENABLED", "")
	client, err := GetMongoClient(context.Background())
	if client != nil || err == nil {
		t.Error("error: client initialized when should fail instead")
	}
	expected := "error: Cannot instantiate Mongo client, error building connection string: [error: no system property found for key: [MONGO_SSL_ENABLED]]"
	actual := fmt.Sprintf("%v", err)
	if expected != actual {
		t.Errorf("error: unexpected error! expected: [%v], actual: [%v]", expected, actual)
	}
}
