package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func TestDlqDocumentsRequireBSONDate(t *testing.T) {
	at := time.Date(2026, 8, 11, 13, 24, 47, 0, time.UTC)
	for _, tc := range []struct {
		name      string
		timestamp interface{}
		wantError bool
	}{
		{"date", at, false},
		{"ISO string", at.Format(time.RFC3339), true},
		{"milliseconds integer", at.UnixMilli(), true},
		{"BSON timestamp", primitive.Timestamp{T: uint32(at.Unix())}, true},
		{"null", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := bson.Marshal(bson.M{"timestamp": tc.timestamp})
			if err != nil {
				t.Fatal(err)
			}
			for _, target := range []interface{}{&ContractDeactivationDlqEntry{}, &ContractDeactivationErrorEvent{}} {
				err := bson.Unmarshal(data, target)
				if (err != nil) != tc.wantError {
					t.Errorf("%T: want error %v, got %v", target, tc.wantError, err)
				}
			}
		})
	}
}

func TestDlqSearchRejectsInvalidPaginationBeforeDatabaseAccess(t *testing.T) {
	repository := &ContractDeactivationDlqRepository{}
	for _, tc := range []struct {
		size  int
		token string
	}{
		{-1, ""}, {0, ""}, {11, ""}, {1, "not-a-token"}, {1, "e30"},
		{1, strings.Repeat("a", 513)},
		{1, "eyJ2IjoxLCJ0IjowLCJpZCI6Im5vdC1hLXV1aWQifQ"},
		{1, base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"t":1000,"id":"00000000-0000-4000-8000-000000000001"}`))},
		{1, base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"t":1000,"id":"00000000-0000-4000-8000-000000000001","extra":true}`))},
		{1, base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"t":1000,"id":"00000000-0000-4000-8000-000000000001"} {}`))},
	} {
		if _, err := repository.Search(context.Background(), tc.size, tc.token); !errors.Is(err, ErrInvalidDlqPagination) {
			t.Errorf("size %d, token %q: want pagination error, got %v", tc.size, tc.token, err)
		}
	}
}

func TestDlqRepositoryIntegration(t *testing.T) {
	uri := os.Getenv("MONGO_INTEGRATION_URI")
	if uri == "" {
		t.Skip("set MONGO_INTEGRATION_URI to run MongoDB integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(cleanup); err != nil {
			t.Error(err)
		}
	})
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Fatal(err)
	}
	db := client.Database("dlq_test_" + uuid.NewString())
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := db.Drop(cleanup); err != nil {
			t.Error(err)
		}
	})
	t.Setenv(PaymentWalletDBNameEnvKey, db.Name())
	t.Setenv(ContractDeactivationDlqCollectionNameEnvKey, "custom-dlq")
	t.Setenv(WalletLogEventsCollectionNameEnvKey, "custom-events")
	repository := NewContractDeactivationDlqRepository(client)
	at := time.Date(2026, 8, 11, 13, 24, 47, 978000000, time.UTC)
	ids := []string{
		"00000000-0000-4000-8000-000000000001",
		"00000000-0000-4000-8000-000000000002",
		"00000000-0000-4000-8000-000000000003",
		"00000000-0000-4000-8000-000000000004",
	}
	_, err = db.Collection("custom-dlq").InsertMany(ctx, []interface{}{
		bson.M{"_id": ids[2], "timestamp": at.Add(time.Second)},
		bson.M{"_id": ids[1], "timestamp": at},
		bson.M{"_id": ids[0], "timestamp": at},
		bson.M{"_id": ids[3], "timestamp": at.Add(2 * time.Second)},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Collection("custom-events").InsertMany(ctx, []interface{}{
		bson.M{"_id": ids[0], "walletId": ids[0], "timestamp": at, "contractId": "contract-1",
			"_class": ContractDeactivationErrorEventClass, "errorType": "COMMUNICATION_ERROR", "attemptNumber": 1},
		bson.M{"_id": ids[1], "walletId": ids[1], "timestamp": at, "contractId": "contract-2",
			"_class": ContractDeactivationErrorEventClass, "errorType": "KO_RESPONSE_RECEIVED",
			"httpErrorCode": 500, "npgErrorCode": "PG_0017", "attemptNumber": 2},
		bson.M{"_id": "unrelated-type", "walletId": ids[0], "_class": "AnotherEvent"},
		bson.M{"_id": ids[3], "walletId": ids[3], "timestamp": at, "contractId": "contract-4",
			"_class": ContractDeactivationErrorEventClass, "errorType": "COMMUNICATION_ERROR", "attemptNumber": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := repository.Search(ctx, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 1 || page.Entries[0].ID != ids[0] || page.NextToken == "" {
		t.Fatalf("unexpected first page: %+v", page)
	}
	if len(page.Events) != 1 || page.Events[0].WalletID != ids[0] || page.Events[0].HTTPErrorCode != nil {
		t.Fatalf("unexpected first-page events: %+v", page.Events)
	}
	// The boundary entry may expire from the DLQ between requests.
	if _, err := db.Collection("custom-dlq").DeleteOne(ctx, bson.M{"_id": ids[0]}); err != nil {
		t.Fatal(err)
	}
	page, err = repository.Search(ctx, 2, page.NextToken)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 2 {
		t.Fatalf("want two entries, got %+v", page)
	}
	got := []string{page.Entries[0].ID, page.Entries[1].ID}
	if !reflect.DeepEqual(got, ids[1:3]) || page.NextToken == "" {
		t.Fatalf("unexpected second page: %+v", page)
	}
	if len(page.Events) != 1 || page.Events[0].HTTPErrorCode == nil || *page.Events[0].HTTPErrorCode != 500 ||
		page.Events[0].NPGErrorCode == nil || *page.Events[0].NPGErrorCode != "PG_0017" {
		t.Fatalf("optional event fields not decoded: %+v", page.Events)
	}
	page, err = repository.Search(ctx, 2, page.NextToken)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 1 || page.Entries[0].ID != ids[3] || page.NextToken != "" {
		t.Fatalf("unexpected final page: %+v", page)
	}
	page, err = repository.Search(ctx, 10, "")
	if err != nil || len(page.Entries) != 3 || page.NextToken != "" {
		t.Fatalf("unexpected page with no lookahead: %+v, %v", page, err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := repository.Search(cancelled, 1, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context cancellation, got %v", err)
	}
	if _, err := db.Collection("custom-dlq").UpdateByID(ctx, ids[1], bson.M{"$set": bson.M{"timestamp": "invalid"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Search(ctx, 10, ""); err == nil {
		t.Fatal("expected malformed DLQ timestamp to fail")
	}
	if _, err := db.Collection("custom-dlq").UpdateByID(ctx, ids[1], bson.M{"$set": bson.M{"timestamp": at}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("custom-events").UpdateByID(ctx, ids[1], bson.M{"$set": bson.M{"httpErrorCode": "invalid"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Search(ctx, 10, ""); err == nil {
		t.Fatal("expected malformed event to fail decoding")
	}
}
