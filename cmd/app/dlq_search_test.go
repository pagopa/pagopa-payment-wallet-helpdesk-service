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

	"github.com/gin-gonic/gin"
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
	validator, err := newRequestValidator("../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(validator)
	api.RegisterHandlers(router, newServerHandler(&Server{
		contractDeactivationDlqService: service.NewContractDeactivationDlqService(client),
	}))
	const path = "/helpdesk/wallets/contractId/deactivations/dlq"
	for _, query := range []string{
		"", "?pageSize=0", "?pageSize=11", "?pageSize=abc",
		"?pageSize=1&searchSequenceToken=invalid",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path+query, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("query %q: want 400, got %d: %s", query, recorder.Code, recorder.Body.String())
		}
		var problem api.ProblemJson
		if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil || problem.Status == nil || *problem.Status != 400 {
			t.Errorf("query %q: want typed 400 problem, got %s", query, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path+"?pageSize=1", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("empty DLQ: want 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
	at := "2026-08-11T13:24:47.978419012Z"
	ids := []string{"00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002"}
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
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
			path+"?pageSize=1&searchSequenceToken="+url.QueryEscape(token), nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("page %d: want 200, got %d: %s", i, recorder.Code, recorder.Body.String())
		}
		var page api.WalletContractIdDlqPageResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Results) != 1 || page.Results[0].Id != id || page.Results[0].ContractId != "contract" {
			t.Fatalf("unexpected page: %+v", page)
		}
		errors := page.Results[0].DeactivationErrors
		if len(errors) != 1 || errors[0].NpgErrorCode == nil || *errors[0].NpgErrorCode != "PG_0017" {
			t.Fatalf("missing NPG error details: %+v", errors)
		}
		if (i == 0 && page.SearchSequenceToken == "") || (i == 1 && page.SearchSequenceToken != "") {
			t.Fatalf("unexpected continuation token on page %d: %q", i, page.SearchSequenceToken)
		}
		token = page.SearchSequenceToken
	}
	if _, err := db.Collection("events").DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path+"?pageSize=1", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("missing events: want 500, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var problem api.ProblemJson
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil || problem.Status == nil || *problem.Status != 500 {
		t.Fatalf("want typed 500 problem, got %s", recorder.Body.String())
	}
}
