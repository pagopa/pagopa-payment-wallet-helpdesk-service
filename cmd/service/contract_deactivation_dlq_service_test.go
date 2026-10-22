package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	repo "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/repository"
)

type dlqRepositoryStub struct {
	page repo.ContractDeactivationDlqPage
	err  error
}

func (r dlqRepositoryStub) Search(context.Context, int, string) (repo.ContractDeactivationDlqPage, error) {
	return r.page, r.err
}

func TestDlqServiceMapsLatestContractAndOrdersAttempts(t *testing.T) {
	at := time.Date(2026, 8, 11, 13, 24, 47, 0, time.UTC)
	httpCode := int32(500)
	npgCode := "PG_0017"
	page := repo.ContractDeactivationDlqPage{
		Entries: []repo.ContractDeactivationDlqEntry{{ID: "wallet-2"}, {ID: "wallet-1"}},
		Events: []repo.ContractDeactivationErrorEvent{
			{ID: "00000000-0000-4000-8000-000000000001", WalletID: "wallet-1", ContractID: "old-contract", Timestamp: at,
				ErrorType: "COMMUNICATION_ERROR", AttemptNumber: 1},
			{ID: "00000000-0000-4000-8000-000000000002", WalletID: "wallet-1", ContractID: "current-contract", Timestamp: at.Add(2 * time.Second),
				ErrorType: "KO_RESPONSE_RECEIVED", AttemptNumber: 2, HTTPErrorCode: &httpCode, NPGErrorCode: &npgCode},
			{ID: "00000000-0000-4000-8000-000000000003", WalletID: "wallet-1", ContractID: "current-contract", Timestamp: at.Add(time.Second),
				ErrorType: "COMMUNICATION_ERROR", AttemptNumber: 1},
			{ID: "00000000-0000-4000-8000-000000000004", WalletID: "wallet-2", ContractID: "other-contract", Timestamp: at,
				ErrorType: "COMMUNICATION_ERROR", AttemptNumber: 1},
		},
		NextToken: "next-page",
	}
	service := &ContractDeactivationDlqService{repository: dlqRepositoryStub{page: page}}
	response, err := service.Search(context.Background(), api.GetWalletContractIdDeactivationDlqRequestObject{
		Params: api.GetWalletContractIdDeactivationDlqParams{PageSize: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	if err := response.VisitGetWalletContractIdDeactivationDlqResponse(recorder); err != nil {
		t.Fatal(err)
	}
	want := `{"continuationCursor":"next-page","results":[{"contractId":"other-contract","deactivationErrors":[{"errorType":"COMMUNICATION_ERROR"}],"id":"wallet-2"},{"contractId":"current-contract","deactivationErrors":[{"errorType":"COMMUNICATION_ERROR"},{"errorType":"KO_RESPONSE_RECEIVED","httpErrorCode":500,"npgErrorCode":"PG_0017"}],"id":"wallet-1"}]}`
	if recorder.Code != 200 || strings.TrimSpace(recorder.Body.String()) != want {
		t.Fatalf("unexpected response %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDlqServiceMapsFailuresAndMalformedEvents(t *testing.T) {
	at := time.Now().UTC()
	valid := repo.ContractDeactivationErrorEvent{
		ID: "00000000-0000-4000-8000-000000000001", WalletID: "wallet", ContractID: "contract", Timestamp: at,
		ErrorType: "COMMUNICATION_ERROR", AttemptNumber: 1,
	}
	for _, tc := range []struct {
		name               string
		page               repo.ContractDeactivationDlqPage
		err                error
		mutate             func(*repo.ContractDeactivationErrorEvent)
		want               int
		wantOmittedDetails bool
		wantEmptyDetails   bool
	}{
		{name: "empty", want: 404},
		{name: "invalid token", err: repo.ErrInvalidDlqPagination, want: 400},
		{name: "database error", err: errors.New("secret database details"), want: 500},
		{name: "missing events", page: repo.ContractDeactivationDlqPage{
			Entries: []repo.ContractDeactivationDlqEntry{{ID: "wallet"}},
		}, want: 200, wantOmittedDetails: true},
		{name: "missing contract", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.ContractID = "" }, want: 200, wantEmptyDetails: true},
		{name: "invalid event ID", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.ID = "not-a-uuid" }, want: 200, wantEmptyDetails: true},
		{name: "malformed stored event", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.Malformed = true }, want: 200, wantEmptyDetails: true},
		{name: "invalid type", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.ErrorType = "invalid" }, want: 200, wantEmptyDetails: true},
		{name: "missing timestamp", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.Timestamp = time.Time{} }, want: 200, wantEmptyDetails: true},
		{name: "invalid attempt", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.AttemptNumber = 0 }, want: 200, wantEmptyDetails: true},
		{name: "invalid HTTP code", mutate: func(e *repo.ContractDeactivationErrorEvent) {
			code := int32(600)
			e.HTTPErrorCode = &code
		}, want: 200, wantEmptyDetails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mutate != nil {
				event := valid
				tc.mutate(&event)
				tc.page = repo.ContractDeactivationDlqPage{
					Entries: []repo.ContractDeactivationDlqEntry{{ID: "wallet"}},
					Events:  []repo.ContractDeactivationErrorEvent{event},
				}
			}
			service := &ContractDeactivationDlqService{repository: dlqRepositoryStub{page: tc.page, err: tc.err}}
			response, err := service.Search(context.Background(), api.GetWalletContractIdDeactivationDlqRequestObject{
				Params: api.GetWalletContractIdDeactivationDlqParams{PageSize: 1},
			})
			if err != nil {
				t.Fatalf("expected typed response, got %v", err)
			}
			recorder := httptest.NewRecorder()
			if err := response.VisitGetWalletContractIdDeactivationDlqResponse(recorder); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != tc.want {
				t.Fatalf("want status %d, got %d: %s", tc.want, recorder.Code, recorder.Body.String())
			}
			if tc.want == 200 {
				var body struct {
					Results []map[string]json.RawMessage `json:"results"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if len(body.Results) != 1 || string(body.Results[0]["id"]) != `"wallet"` {
					t.Fatalf("expected wallet to remain in successful response: %s", recorder.Body.String())
				}
				if tc.wantOmittedDetails && (body.Results[0]["contractId"] != nil || body.Results[0]["deactivationErrors"] != nil) {
					t.Errorf("expected absent detail fields, got %s", recorder.Body.String())
				}
				if tc.wantEmptyDetails && (string(body.Results[0]["contractId"]) != `""` || string(body.Results[0]["deactivationErrors"]) != `[]`) {
					t.Errorf("expected empty detail fields, got %s", recorder.Body.String())
				}
			} else if !strings.Contains(recorder.Body.String(), `"status":`) {
				t.Errorf("want problem response %d, got %s", tc.want, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "secret database details") {
				t.Error("response exposed internal error")
			}
		})
	}
}
