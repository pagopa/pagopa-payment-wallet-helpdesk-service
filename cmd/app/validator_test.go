package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
)

func TestRequestValidatorAcceptsDifferentHostsAndValidatesParameters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.yaml")
	spec := `openapi: 3.0.3
info:
  title: Validator regression test
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /items:
    get:
      parameters:
        - in: query
          name: limit
          schema:
            type: integer
            minimum: 1
      responses:
        '204':
          description: Success
`
	if err := os.WriteFile(path, []byte(spec), 0600); err != nil {
		t.Fatal(err)
	}
	validator, err := newRequestValidator(path)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(validator)
	router.GET("/items", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for _, tc := range []struct {
		name string
		url  string
		want int
	}{
		{"localhost", "http://localhost:8080/items?limit=1", http.StatusNoContent},
		{"probe", "http://10.0.0.1:8080/items?limit=1", http.StatusNoContent},
		{"production", "https://api.example.com/items?limit=1", http.StatusNoContent},
		{"invalid type", "http://localhost:8080/items?limit=invalid", http.StatusBadRequest},
		{"invalid range", "http://localhost:8080/items?limit=0", http.StatusBadRequest},
		{"unknown path", "http://localhost:8080/unknown", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.url, nil))
			if recorder.Code != tc.want {
				t.Errorf("status = %d, want %d; body = %s", recorder.Code, tc.want, recorder.Body.String())
			}
		})
	}
}

func TestRequestValidatorAllowsPublicServiceInfo(t *testing.T) {
	validator, err := newRequestValidator("../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(validator)
	api.RegisterHandlers(router, newServerHandler(&Server{}))

	for _, url := range []string{"http://localhost:8080/info", "http://10.0.0.1:8080/info"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, url, nil))
		if recorder.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200; body = %s", url, recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"version":`) {
			t.Errorf("%s: missing service version in response: %s", url, recorder.Body.String())
		}
	}
}

func TestRequestValidatorPreservesWalletSecurityRequirements(t *testing.T) {
	validator, err := newRequestValidator("../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(validator)
	router.GET("/helpdesk/wallets/:userId", func(c *gin.Context) {
		t.Error("request without configured authentication reached handler")
		c.Status(http.StatusOK)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://localhost:8080/helpdesk/wallets/12345", nil))

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "SecurityRequirementsError") {
		t.Errorf("want authentication validation error, got status %d; body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRequestValidatorRejectsUnreadableAndInvalidSpecifications(t *testing.T) {
	dir := t.TempDir()
	invalidPath := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte("not: [valid"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dir, "missing.yaml"), invalidPath} {
		if _, err := newRequestValidator(path); err == nil {
			t.Errorf("expected specification loading error for %s", path)
		}
	}
}
