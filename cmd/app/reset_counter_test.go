package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
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

func TestResetCounterRequestValidation(t *testing.T) {
	validator, err := newRequestValidator("../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(validator)
	router.PATCH("/helpdesk/wallets/:walletId/deactivationRetryCount", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	for _, tc := range []struct {
		id   string
		want int
	}{
		{"00000000-0000-4000-8000-000000000001", http.StatusNoContent},
		{"not-a-uuid", http.StatusBadRequest},
		{"00000000000040008000000000000001", http.StatusBadRequest},
		{"00000000-0000-4000-8000-00000000000A", http.StatusBadRequest},
	} {
		t.Run(tc.id, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPatch,
				"/helpdesk/wallets/"+tc.id+"/deactivationRetryCount", nil))
			if recorder.Code != tc.want {
				t.Fatalf("want %d, got %d: %s", tc.want, recorder.Code, recorder.Body.String())
			}
			if tc.want >= 400 {
				assertResetProblem(t, recorder, tc.want)
			}
		})
	}
}

func TestResetCounterHTTPIntegration(t *testing.T) {
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
	db := client.Database("reset_http_test_" + uuid.NewString())
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
	t.Setenv(repo.PaymentWalletWalletsCollectionNameEnvKey, "payment-wallets")
	router := resetCounterRouter(t, client)
	wallets := db.Collection("payment-wallets")
	const otherID = "00000000-0000-4000-8000-000000000099"
	other := bson.M{"_id": otherID, "deactivationRetryCount": int32(7)}
	if _, err := wallets.InsertOne(ctx, other); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		status string
		count  interface{}
		inDLQ  bool
	}{
		{"positive count", "VALIDATED", int32(5), true},
		{"already zero", "DELETED", int32(0), true},
		{"missing field", "ERROR", nil, true},
		{"not in DLQ", "VALIDATED", int32(3), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := uuid.NewString()
			document := bson.M{
				"_id": id, "userId": "user", "status": tc.status,
				"contractId": "contract", "details": bson.M{"untouched": "value"},
			}
			if tc.count != nil {
				document["deactivationRetryCount"] = tc.count
			}
			if _, err := wallets.InsertOne(ctx, document); err != nil {
				t.Fatal(err)
			}
			unauthorized := httptest.NewRecorder()
			router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPatch,
				"/helpdesk/wallets/"+id+"/deactivationRetryCount", nil))
			assertResetProblem(t, unauthorized, http.StatusUnauthorized)
			var unchanged bson.M
			if err := wallets.FindOne(ctx, bson.M{"_id": id}).Decode(&unchanged); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(unchanged, document) {
				t.Fatalf("unauthorized reset modified the wallet: %#v", unchanged)
			}
			dlq := bson.M{"_id": id, "timestamp": "2026-08-11T13:24:47Z"}
			if tc.inDLQ {
				if _, err := db.Collection("wallet-contractId-deactivation-dlq").InsertOne(ctx, dlq); err != nil {
					t.Fatal(err)
				}
			}
			for attempt := 0; attempt < 2; attempt++ {
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, authenticatedRequest(http.MethodPatch,
					"/helpdesk/wallets/"+id+"/deactivationRetryCount"))
				if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
					t.Fatalf("attempt %d: want empty 204, got %d: %s", attempt, recorder.Code, recorder.Body.String())
				}
			}
			var got bson.M
			if err := wallets.FindOne(ctx, bson.M{"_id": id}).Decode(&got); err != nil {
				t.Fatal(err)
			}
			document["deactivationRetryCount"] = int32(0)
			if !reflect.DeepEqual(got, document) {
				t.Fatalf("wallet = %#v, want %#v", got, document)
			}
			var gotDLQ bson.M
			err := db.Collection("wallet-contractId-deactivation-dlq").FindOne(ctx, bson.M{"_id": id}).Decode(&gotDLQ)
			if !tc.inDLQ {
				if err != mongo.ErrNoDocuments {
					t.Fatalf("reset must not create a DLQ entry: %v, %#v", err, gotDLQ)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotDLQ, dlq) {
				t.Fatalf("DLQ was modified: %#v", gotDLQ)
			}
		})
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, authenticatedRequest(http.MethodPatch,
		"/helpdesk/wallets/00000000-0000-4000-8000-000000000098/deactivationRetryCount"))
	assertResetProblem(t, recorder, http.StatusNotFound)
	count, err := wallets.CountDocuments(ctx, bson.M{"_id": "00000000-0000-4000-8000-000000000098"})
	if err != nil || count != 0 {
		t.Fatalf("missing wallet must not be created: count=%d, error=%v", count, err)
	}
	var gotOther bson.M
	if err := wallets.FindOne(ctx, bson.M{"_id": otherID}).Decode(&gotOther); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotOther, other) {
		t.Fatalf("unrelated wallet was modified: %#v", gotOther)
	}
}

func TestResetCounterDatabaseFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Disconnect(ctx); err != nil {
		t.Fatal(err)
	}
	router := resetCounterRouter(t, client)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, authenticatedRequest(http.MethodPatch,
		"/helpdesk/wallets/00000000-0000-4000-8000-000000000001/deactivationRetryCount"))
	assertResetProblem(t, recorder, http.StatusInternalServerError)
	if strings.Contains(recorder.Body.String(), "disconnected") {
		t.Fatalf("internal database error leaked: %s", recorder.Body.String())
	}
}

func resetCounterRouter(t *testing.T, client *mongo.Client) *gin.Engine {
	t.Helper()
	setTestAPIKeys(t)
	router, err := newRouter(&Server{
		paymentWalletHelpdeskService: service.NewPaymentWalletHelpdeskService(client),
	}, "../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return router
}

func assertResetProblem(t *testing.T, recorder *httptest.ResponseRecorder, status int) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("want %d, got %d: %s", status, recorder.Code, recorder.Body.String())
	}
	var problem api.ProblemJson
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil ||
		problem.Status == nil || int(*problem.Status) != status ||
		problem.Title == nil || *problem.Title == "" || problem.Detail == nil || *problem.Detail == "" {
		t.Fatalf("want typed %d problem, got %s", status, recorder.Body.String())
	}
}
