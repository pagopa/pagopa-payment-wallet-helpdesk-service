package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

func TestDlqDocumentsRequireStringTimestamp(t *testing.T) {
	at := time.Date(2026, 8, 11, 13, 24, 47, 978419012, time.UTC)
	for _, tc := range []struct {
		name               string
		timestamp          interface{}
		wantEntryError     bool
		wantEventMalformed bool
		wantEventTimestamp bool
	}{
		{name: "date", timestamp: at, wantEntryError: true, wantEventMalformed: true},
		{name: "ISO string", timestamp: at.Format(time.RFC3339Nano), wantEventTimestamp: true},
		{name: "ISO offset", timestamp: "2026-08-11T15:24:47.978419012+02:00", wantEventTimestamp: true},
		{name: "invalid string", timestamp: "invalid", wantEventMalformed: true},
		{name: "empty string", timestamp: "", wantEventMalformed: true},
		{name: "milliseconds integer", timestamp: at.UnixMilli(), wantEntryError: true, wantEventMalformed: true},
		{name: "BSON timestamp", timestamp: primitive.Timestamp{T: uint32(at.Unix())}, wantEntryError: true, wantEventMalformed: true},
		{name: "null", timestamp: nil, wantEntryError: true, wantEventMalformed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := bson.Marshal(bson.M{
				"_id":           "00000000-0000-4000-8000-000000000001",
				"walletId":      "00000000-0000-4000-8000-000000000001",
				"timestamp":     tc.timestamp,
				"contractId":    "contract",
				"errorType":     "COMMUNICATION_ERROR",
				"attemptNumber": 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			var entry ContractDeactivationDlqEntry
			if err := bson.Unmarshal(data, &entry); (err != nil) != tc.wantEntryError {
				t.Errorf("DLQ entry: want error %v, got %v", tc.wantEntryError, err)
			}
			if err := bson.Unmarshal(data, &entry); err == nil && entry.Timestamp != tc.timestamp {
				t.Errorf("DLQ timestamp = %q, want %q", entry.Timestamp, tc.timestamp)
			}
			var event ContractDeactivationErrorEvent
			if err := bson.Unmarshal(data, &event); err != nil {
				t.Fatalf("event decoding must preserve malformed rows for per-wallet handling: %v", err)
			}
			if event.Malformed != tc.wantEventMalformed {
				t.Errorf("event malformed = %v, want %v", event.Malformed, tc.wantEventMalformed)
			}
			if tc.wantEventTimestamp && !event.Timestamp.Equal(at) {
				t.Errorf("event timestamp lost precision or offset: %s", event.Timestamp)
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
		{1, base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"t":1000,"id":"00000000-0000-4000-8000-000000000001"}`))},
		{1, base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"t":"2026-08-11T13:24:47Z","id":"00000000-0000-4000-8000-000000000001"}`))},
		{1, base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"t":"2026-08-11T13:24:47Z","id":"00000000-0000-4000-8000-000000000001","extra":true}`))},
		{1, base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"t":"2026-08-11T13:24:47Z","id":"00000000-0000-4000-8000-000000000001"} {}`))},
		{1, base64.RawURLEncoding.EncodeToString([]byte(`{"v":3,"id":"00000000-0000-4000-8000-000000000001"}`))},
	} {
		if _, err := repository.Search(context.Background(), tc.size, tc.token); !errors.Is(err, ErrInvalidDlqPagination) {
			t.Errorf("size %d, token %q: want pagination error, got %v", tc.size, tc.token, err)
		}
	}
}

func TestDecodeDlqSequenceTokenAcceptsNonRFC3339Timestamp(t *testing.T) {
	const timestamp = "not-a-timestamp"
	data, err := json.Marshal(dlqSequenceToken{
		Version: 3, Timestamp: timestamp, ID: "wallet-id",
	})
	if err != nil {
		t.Fatal(err)
	}
	token, err := decodeDlqSequenceToken(base64.RawURLEncoding.EncodeToString(data))
	if err != nil {
		t.Fatalf("non-RFC3339 string timestamp should be accepted in the cursor: %v", err)
	}
	if token.Timestamp != timestamp {
		t.Fatalf("cursor timestamp = %q, want %q", token.Timestamp, timestamp)
	}
}

func TestDecodeDlqSequenceTokenAcceptsNonUUIDWalletID(t *testing.T) {
	const walletID = "wallet-id-not-a-uuid"
	data, err := json.Marshal(dlqSequenceToken{
		Version: 3, Timestamp: "2026-08-11T13:24:47Z", ID: walletID,
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	token, err := decodeDlqSequenceToken(encoded)
	if err != nil {
		t.Fatalf("non-UUID wallet ID should be accepted in the continuation cursor: %v", err)
	}
	if token.ID != walletID {
		t.Fatalf("cursor wallet ID = %q, want %q", token.ID, walletID)
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
	at := "2026-08-11T13:24:47.978419012Z"
	ids := []string{
		"wallet-id-not-a-uuid",
		"00000000-0000-4000-8000-000000000002",
		"00000000-0000-4000-8000-000000000003",
		"00000000-0000-4000-8000-000000000004",
	}
	_, err = db.Collection("custom-dlq").InsertMany(ctx, []interface{}{
		bson.M{"_id": ids[2], "timestamp": at},
		bson.M{"_id": ids[1], "timestamp": at},
		bson.M{"_id": ids[0], "timestamp": "2026-08-11T13:24:47.978419011Z"},
		bson.M{"_id": ids[3], "timestamp": "2026-08-11T13:24:48.000000000Z"},
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
	if len(page.Entries) != 1 || page.Entries[0].ID != ids[3] || page.NextToken == "" {
		t.Fatalf("unexpected first page: %+v", page)
	}
	if len(page.Events) != 1 || page.Events[0].WalletID != ids[3] || page.Events[0].HTTPErrorCode != nil {
		t.Fatalf("unexpected first-page events: %+v", page.Events)
	}
	token, err := decodeDlqSequenceToken(page.NextToken)
	if err != nil || token.Version != 3 || token.Timestamp != "2026-08-11T13:24:48.000000000Z" {
		t.Fatalf("cursor must preserve the stored timestamp exactly: %+v, %v", token, err)
	}
	// The boundary entry may expire from the DLQ between requests.
	if _, err := db.Collection("custom-dlq").DeleteOne(ctx, bson.M{"_id": ids[3]}); err != nil {
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
	if !reflect.DeepEqual(got, []string{ids[2], ids[1]}) || page.NextToken == "" {
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
	if len(page.Entries) != 1 || page.Entries[0].ID != ids[0] || page.NextToken != "" {
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
	wantOrder := []string{ids[1], ids[2], ids[0]}
	var gotOrder []string
	cursorToken := ""
	for {
		page, err = repository.Search(ctx, 1, cursorToken)
		if err != nil {
			t.Fatalf("malformed string timestamp should not prevent page traversal: %v", err)
		}
		if len(page.Entries) != 1 {
			t.Fatalf("expected one entry on each page, got %+v", page.Entries)
		}
		gotOrder = append(gotOrder, page.Entries[0].ID)
		if page.Entries[0].ID == ids[1] && page.Entries[0].Timestamp != "invalid" {
			t.Fatalf("malformed timestamp was not preserved: %+v", page.Entries[0])
		}
		if page.NextToken == "" {
			break
		}
		cursorToken = page.NextToken
	}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("unexpected traversal order with malformed timestamp: got %v, want %v", gotOrder, wantOrder)
	}
	if _, err := db.Collection("custom-dlq").UpdateByID(ctx, ids[1], bson.M{"$set": bson.M{"timestamp": at}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("custom-events").UpdateByID(ctx, ids[1], bson.M{"$set": bson.M{"timestamp": "invalid"}}); err != nil {
		t.Fatal(err)
	}
	page, err = repository.Search(ctx, 10, "")
	malformedEventFound := false
	for _, event := range page.Events {
		if event.WalletID == ids[1] && event.Malformed {
			malformedEventFound = true
		}
	}
	if err != nil || !malformedEventFound {
		t.Fatalf("malformed event timestamp should remain available for per-wallet handling: %+v, %v", page.Events, err)
	}
	if _, err := db.Collection("custom-events").UpdateByID(ctx, ids[1], bson.M{"$set": bson.M{"timestamp": at}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("custom-events").UpdateByID(ctx, ids[1], bson.M{"$set": bson.M{"httpErrorCode": "invalid"}}); err != nil {
		t.Fatal(err)
	}
	page, err = repository.Search(ctx, 10, "")
	malformedEventFound = false
	for _, event := range page.Events {
		if event.WalletID == ids[1] && event.Malformed {
			malformedEventFound = true
		}
	}
	if err != nil || !malformedEventFound {
		t.Fatalf("malformed event should remain associated with its wallet: %+v, %v", page.Events, err)
	}
	if _, err := db.Collection("custom-events").UpdateByID(ctx, ids[1], bson.M{
		"$set": bson.M{"httpErrorCode": 500, "timestamp": "invalid"},
	}); err != nil {
		t.Fatal(err)
	}
	page, err = repository.Search(ctx, 10, "")
	malformedEventFound = false
	for _, event := range page.Events {
		if event.WalletID == ids[1] && event.Malformed {
			malformedEventFound = true
		}
	}
	if err != nil || !malformedEventFound {
		t.Fatalf("malformed event timestamp should not fail page decoding: %+v, %v", page.Events, err)
	}
}
