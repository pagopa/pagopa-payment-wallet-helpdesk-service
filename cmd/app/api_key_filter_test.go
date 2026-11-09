package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
)

func setTestAPIKeys(t *testing.T) {
	t.Helper()
	t.Setenv("SECURITY_API_KEYS_SECURED_PATHS", "/helpdesk/")
	t.Setenv("SECURITY_API_KEYS_PRIMARY", "test-primary")
	t.Setenv("SECURITY_API_KEYS_SECONDARY", "test-secondary")
}

func authenticatedRequest(method, path string) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("x-api-key", "test-primary")
	return request
}

func TestAPIKeyFilterProtectsWalletOperations(t *testing.T) {
	setTestAPIKeys(t)
	server := &authenticatedTestServer{}
	router, err := newRouter(server, "../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/helpdesk/wallets/user", http.StatusOK},
		{http.MethodGet, "/helpdesk/wallets/contractId/deactivations/dlq?pageSize=1", http.StatusNotFound},
		{http.MethodPatch, "/helpdesk/wallets/00000000-0000-4000-8000-000000000001/deactivationRetryCount", http.StatusNoContent},
	} {
		for _, tc := range []struct {
			name string
			keys []string
			want int
		}{
			{"missing", nil, http.StatusUnauthorized},
			{"empty", []string{""}, http.StatusUnauthorized},
			{"invalid", []string{"test-invalid"}, http.StatusUnauthorized},
			{"primary", []string{"test-primary"}, operation.want},
			{"secondary", []string{"test-secondary"}, operation.want},
			{"case sensitive", []string{"TEST-PRIMARY"}, http.StatusUnauthorized},
			{"not trimmed", []string{" test-primary "}, http.StatusUnauthorized},
			{"combined keys", []string{"test-primary,test-secondary"}, http.StatusUnauthorized},
			{"duplicate headers", []string{"test-primary", "test-invalid"}, http.StatusUnauthorized},
		} {
			t.Run(operation.method+operation.path+"/"+tc.name, func(t *testing.T) {
				callsBefore := server.calls
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(operation.method, operation.path, nil)
				for _, key := range tc.keys {
					request.Header.Add("x-api-key", key)
				}
				router.ServeHTTP(recorder, request)
				if recorder.Code != tc.want {
					t.Fatalf("want %d, got %d: %s", tc.want, recorder.Code, recorder.Body.String())
				}
				if tc.want == http.StatusUnauthorized {
					if server.calls != callsBefore {
						t.Fatal("unauthorized request reached the handler")
					}
					var problem api.ProblemJson
					if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil ||
						problem.Status == nil || *problem.Status != 401 ||
						problem.Title == nil || *problem.Title != "Unauthorized" ||
						problem.Detail == nil || *problem.Detail == "" {
						t.Fatalf("want 401 ProblemJson, got %s", recorder.Body.String())
					}
					if recorder.Header().Get("Content-Type") != "application/json; charset=utf-8" {
						t.Fatalf("unexpected content type: %s", recorder.Header().Get("Content-Type"))
					}
				} else if server.calls != callsBefore+1 {
					t.Fatal("authorized request did not reach the handler exactly once")
				}
			})
		}
	}
}

func TestAPIKeyFilterRunsBeforeSchemaValidation(t *testing.T) {
	setTestAPIKeys(t)
	router, err := newRouter(&authenticatedTestServer{}, "../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/helpdesk/wallets/user?status=INVALID",
		"/helpdesk/wallets/contractId/deactivations/dlq?pageSize=0",
		"/helpdesk/wallets/not-a-uuid/deactivationRetryCount",
	} {
		for _, key := range []string{"", "test-primary"} {
			method := http.MethodGet
			if strings.Contains(path, "deactivationRetryCount") {
				method = http.MethodPatch
			}
			request := httptest.NewRequest(method, path, nil)
			request.Header.Set("x-api-key", key)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			want := http.StatusUnauthorized
			if key != "" {
				want = http.StatusBadRequest
			}
			if recorder.Code != want {
				t.Errorf("%s with key %q: want %d, got %d: %s", path, key, want, recorder.Code, recorder.Body.String())
			}
		}
	}
}

func TestAPIKeyFilterAllowsServiceInfo(t *testing.T) {
	setTestAPIKeys(t)
	// Even a root prefix must not block the public Kubernetes probe.
	t.Setenv("SECURITY_API_KEYS_SECURED_PATHS", "/")
	router, err := newRouter(&Server{}, "../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "test-invalid", "test-primary"} {
		request := httptest.NewRequest(http.MethodGet, "/info", nil)
		request.Header.Set("x-api-key", key)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"version":`) {
			t.Fatalf("want public service info, got %d: %s", recorder.Code, recorder.Body.String())
		}
	}
}

func TestAPIKeyFilterMatchesConfiguredPrefixes(t *testing.T) {
	setTestAPIKeys(t)
	t.Setenv("SECURITY_API_KEYS_SECURED_PATHS", " /helpdesk/ , /internal/ ")
	filter, err := newAPIKeyFilter()
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(filter)
	router.GET("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, tc := range []struct {
		path string
		want int
	}{
		{"/helpdesk/wallets/user", http.StatusUnauthorized},
		{"/internal/subpath", http.StatusUnauthorized},
		{"/public", http.StatusNoContent},
		{"/helpdesk-other", http.StatusNoContent},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if recorder.Code != tc.want {
			t.Errorf("%s: want %d, got %d", tc.path, tc.want, recorder.Code)
		}
	}
}

func TestAPIKeyConfigurationFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{"SECURITY_API_KEYS_PRIMARY", ""},
		{"SECURITY_API_KEYS_PRIMARY", " \t "},
		{"SECURITY_API_KEYS_SECONDARY", ""},
		{"SECURITY_API_KEYS_SECONDARY", " \t "},
		{"SECURITY_API_KEYS_SECURED_PATHS", ""},
		{"SECURITY_API_KEYS_SECURED_PATHS", " \t "},
		{"SECURITY_API_KEYS_SECURED_PATHS", "helpdesk"},
		{"SECURITY_API_KEYS_SECURED_PATHS", "/helpdesk/,"},
		{"SECURITY_API_KEYS_SECURED_PATHS", "/helpdesk/, ,/internal/"},
		{"SECURITY_API_KEYS_SECURED_PATHS", "/helpdesk/?key=test-primary"},
		{"SECURITY_API_KEYS_SECURED_PATHS", "/helpdesk/#fragment"},
	} {
		t.Run(tc.name+"/"+tc.value, func(t *testing.T) {
			setTestAPIKeys(t)
			t.Setenv(tc.name, tc.value)
			router, err := newRouter(&Server{}, "../../api-spec/api.yaml")
			if err == nil || router != nil {
				t.Fatalf("invalid configuration must prevent router creation: %v", err)
			}
			if !strings.Contains(err.Error(), tc.name) {
				t.Fatalf("configuration error must identify %s: %v", tc.name, err)
			}
			if strings.Contains(err.Error(), "test-primary") || strings.Contains(err.Error(), "test-secondary") {
				t.Fatalf("configuration error leaked a key: %v", err)
			}
		})
	}
}

func TestAPIKeyFilterDoesNotLogSecrets(t *testing.T) {
	setTestAPIKeys(t)
	var logs bytes.Buffer
	writer := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(writer) })
	filter, err := newAPIKeyFilter()
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(filter)
	router.GET("/helpdesk/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, key := range []string{"test-primary", "test-secondary", "test-invalid"} {
		request := httptest.NewRequest(http.MethodGet, "/helpdesk/protected", nil)
		request.Header.Set("x-api-key", key)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if strings.Contains(recorder.Body.String(), key) {
			t.Fatalf("key leaked in response: %s", recorder.Body.String())
		}
	}
	if !strings.Contains(logs.String(), "Unauthorized") {
		t.Fatalf("authentication failure was not logged: %s", logs.String())
	}
	for _, key := range []string{"test-primary", "test-secondary", "test-invalid"} {
		if strings.Contains(logs.String(), key) {
			t.Fatalf("key leaked in logs: %s", logs.String())
		}
	}
}

func TestAPIKeysAreNotLoggedOnHandlerPanic(t *testing.T) {
	setTestAPIKeys(t)
	var logs bytes.Buffer
	mode, errorWriter, writer := gin.Mode(), gin.DefaultErrorWriter, log.Writer()
	gin.SetMode(gin.DebugMode)
	gin.DefaultErrorWriter = &logs
	log.SetOutput(&logs)
	t.Cleanup(func() {
		gin.SetMode(mode)
		gin.DefaultErrorWriter = errorWriter
		log.SetOutput(writer)
	})
	router, err := newRouter(&panickingAPIKeyTestServer{}, "../../api-spec/api.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"test-primary", "test-secondary"} {
		request := httptest.NewRequest(http.MethodGet, "/helpdesk/wallets/user", nil)
		request.Header.Set("x-api-key", key)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("want 500 after panic, got %d", recorder.Code)
		}
		if strings.Contains(logs.String(), key) || strings.Contains(recorder.Body.String(), key) {
			t.Fatal("API key leaked during panic recovery")
		}
	}
	if logs.Len() == 0 {
		t.Fatal("panic was not logged")
	}
}

type panickingAPIKeyTestServer struct {
	Server
}

func (*panickingAPIKeyTestServer) GetWallets(context.Context, api.GetWalletsRequestObject) (api.GetWalletsResponseObject, error) {
	panic("test handler panic")
}

type authenticatedTestServer struct {
	Server
	calls int
}

func (s *authenticatedTestServer) GetWallets(context.Context, api.GetWalletsRequestObject) (api.GetWalletsResponseObject, error) {
	s.calls++
	return api.GetWallets200JSONResponse{Wallets: []api.WalletDetail{}}, nil
}

//lint:ignore ST1003 Method name must match the generated OpenAPI interface.
func (s *authenticatedTestServer) GetWalletContractIdDeactivationDlq(context.Context, api.GetWalletContractIdDeactivationDlqRequestObject) (api.GetWalletContractIdDeactivationDlqResponseObject, error) {
	s.calls++
	return api.GetWalletContractIdDeactivationDlq404JSONResponse{}, nil
}

func (s *authenticatedTestServer) ResetWalletDeactivationRetryCount(context.Context, api.ResetWalletDeactivationRetryCountRequestObject) (api.ResetWalletDeactivationRetryCountResponseObject, error) {
	s.calls++
	return api.ResetWalletDeactivationRetryCount204Response{}, nil
}
