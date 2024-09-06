package cosmosdb

import (
	"os"
	"testing"
)

func TestGetMongoClientShouldReturnClientSuccessfully(t *testing.T) {
	os.Clearenv()
	t.Setenv("MONGO_USERNAME", "username")
	t.Setenv("MONGO_PASSWORD", "password")
	t.Setenv("MONGO_HOSTNAME", "localhost")
	t.Setenv("MONGO_PORT", "1234")
	t.Setenv("MONGO_DB_NAME", "dbName")
	t.Setenv("MONGO_SSL_ENABLED", "false")
	client, err := GetMongoClient()
	if err != nil {
		t.FailNow()
	}
	if client == nil {
		t.FailNow()
	}
}
