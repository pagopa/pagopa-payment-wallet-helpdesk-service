package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	repo "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/repository"
	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/service"
)

func TestDlqSearchHTTPIntegration(t *testing.T) {
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
	db := client.Database("dlq_http_test_" + uuid.NewString())
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := db.Drop(cleanup); err != nil {
			t.Error(err)
		}
		if err := client.Disconnect(cleanup); err != nil {
			t.Error(err)
		}
	})
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Fatal(err)
	}
	t.Setenv(repo.PaymentWalletDBNameEnvKey, db.Name())
	t.Setenv(repo.ContractDeactivationDlqCollectionNameEnvKey, "dlq")
	t.Setenv(repo.WalletLogEventsCollectionNameEnvKey, "events")
	setTestAPIKeys(t)
	router, err := newRouter(&Server{
		contractDeactivationDlqService: service.NewContractDeactivationDlqService(client),
	}, "../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	const path = "/helpdesk/wallets/contractId/deactivations/dlq"
	for _, query := range []string{
		"", "?pageSize=0", "?pageSize=11", "?pageSize=abc",
		"?pageSize=1&continuationCursor=invalid",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, authenticatedRequest(http.MethodGet, path+query))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("query %q: want 400, got %d: %s", query, recorder.Code, recorder.Body.String())
		}
		var problem api.ProblemJson
		if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil || problem.Status == nil || *problem.Status != 400 {
			t.Errorf("query %q: want typed 400 problem, got %s", query, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, authenticatedRequest(http.MethodGet, path+"?pageSize=1"))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("empty DLQ: want 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
	at := "2026-08-11T13:24:47.978419012Z"
	ids := []string{"00000000-0000-4000-8000-000000000002", "00000000-0000-4000-8000-000000000001"}
	for _, id := range ids {
		if _, err := db.Collection("dlq").InsertOne(ctx, bson.M{"_id": id, "timestamp": at}); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Collection("events").InsertOne(ctx, bson.M{
			"_id": uuid.NewString(), "walletId": id, "contractId": "contract",
			"timestamp": at, "_class": repo.ContractDeactivationErrorEventClass,
			"attemptNumber": 1, "errorType": "KO_RESPONSE_RECEIVED", "httpErrorCode": 500, "npgErrorCode": "PG_0017",
		}); err != nil {
			t.Fatal(err)
		}
	}
	token := ""
	for i, id := range ids {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, authenticatedRequest(http.MethodGet,
			path+"?pageSize=1&continuationCursor="+url.QueryEscape(token)))
		if recorder.Code != http.StatusOK {
			t.Fatalf("page %d: want 200, got %d: %s", i, recorder.Code, recorder.Body.String())
		}
		var page api.WalletContractIdDlqPageResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Results) != 1 || page.Results[0].Id != id || page.Results[0].ContractId == nil || *page.Results[0].ContractId != "contract" {
			t.Fatalf("unexpected page: %+v", page)
		}
		errors := page.Results[0].DeactivationErrors
		if errors == nil || len(*errors) != 1 || (*errors)[0].NpgErrorCode == nil || *(*errors)[0].NpgErrorCode != "PG_0017" {
			t.Fatalf("missing NPG error details: %+v", errors)
		}
		if (i == 0 && page.ContinuationCursor == "") || (i == 1 && page.ContinuationCursor != "") {
			t.Fatalf("unexpected continuation cursor on page %d: %q", i, page.ContinuationCursor)
		}
		token = page.ContinuationCursor
	}
	if _, err := db.Collection("events").UpdateOne(ctx, bson.M{"walletId": ids[0]}, bson.M{
		"$set": bson.M{"errorType": "UNRECOGNIZED_ERROR"},
	}); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, authenticatedRequest(http.MethodGet, path+"?pageSize=1"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("malformed event: want 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var malformedPage struct {
		Results []map[string]json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &malformedPage); err != nil {
		t.Fatal(err)
	}
	if len(malformedPage.Results) != 1 || malformedPage.Results[0]["contractId"] != nil ||
		malformedPage.Results[0]["deactivationErrors"] != nil {
		t.Fatalf("malformed event details should be omitted: %s", recorder.Body.String())
	}
	if _, err := db.Collection("events").UpdateOne(ctx, bson.M{"walletId": ids[0]}, bson.M{
		"$set": bson.M{"errorType": "KO_RESPONSE_RECEIVED", "timestamp": "not-a-timestamp"},
	}); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, authenticatedRequest(http.MethodGet, path+"?pageSize=1"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("malformed event timestamp: want 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &malformedPage); err != nil {
		t.Fatal(err)
	}
	if len(malformedPage.Results) != 1 || malformedPage.Results[0]["contractId"] != nil ||
		malformedPage.Results[0]["deactivationErrors"] != nil {
		t.Fatalf("malformed timestamp details should be omitted: %s", recorder.Body.String())
	}
	if _, err := db.Collection("events").UpdateOne(ctx, bson.M{"walletId": ids[0]}, bson.M{
		"$set": bson.M{"timestamp": at},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("events").DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, authenticatedRequest(http.MethodGet, path+"?pageSize=1"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("missing events: want 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var noEventsPage struct {
		Results []map[string]json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &noEventsPage); err != nil {
		t.Fatal(err)
	}
	if len(noEventsPage.Results) != 1 || noEventsPage.Results[0]["contractId"] != nil ||
		noEventsPage.Results[0]["deactivationErrors"] != nil {
		t.Fatalf("missing events should omit optional details: %s", recorder.Body.String())
	}
}
