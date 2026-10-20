package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
)

func TestServerHandlerReturnsServiceInfo(t *testing.T) {
	router := gin.New()
	api.RegisterHandlers(router, newServerHandler(&Server{}))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/info", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid service info JSON: %v", err)
	}
	if len(body) != 1 || body["version"] != version {
		t.Errorf("body = %v, want only the embedded service version", body)
	}
}

func TestServerHandlerKeepsInternalErrorsOutOfResponse(t *testing.T) {
	internalError := errors.New("database connection failed")
	router := gin.New()
	var recordedError error
	router.Use(func(c *gin.Context) {
		c.Next()
		if lastError := c.Errors.Last(); lastError != nil {
			recordedError = lastError.Err
		}
	})
	api.RegisterHandlers(router, newServerHandler(&failingWalletServer{err: internalError}))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/helpdesk/wallets/user-id", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("body = %q, want empty response on internal error", recorder.Body.String())
	}
	if !errors.Is(recordedError, internalError) {
		t.Errorf("recorded error = %v, want %v", recordedError, internalError)
	}
}

type failingWalletServer struct {
	Server
	err error
}

func (s *failingWalletServer) GetWallets(context.Context, api.GetWalletsRequestObject) (api.GetWalletsResponseObject, error) {
	return nil, s.err
}
