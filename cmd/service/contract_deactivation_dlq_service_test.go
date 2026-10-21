package service

import (
	"context"
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
	want := `{"results":[{"contractId":"other-contract","deactivationErrors":[{"errorType":"COMMUNICATION_ERROR"}],"id":"wallet-2"},{"contractId":"current-contract","deactivationErrors":[{"errorType":"COMMUNICATION_ERROR"},{"errorType":"KO_RESPONSE_RECEIVED","httpErrorCode":500,"npgErrorCode":"PG_0017"}],"id":"wallet-1"}],"searchSequenceToken":"next-page"}`
	if recorder.Code != 200 || strings.TrimSpace(recorder.Body.String()) != want {
		t.Fatalf("unexpected response %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDlqServiceMapsFailuresToProblemResponses(t *testing.T) {
	at := time.Now().UTC()
	valid := repo.ContractDeactivationErrorEvent{
		ID: "00000000-0000-4000-8000-000000000001", WalletID: "wallet", ContractID: "contract", Timestamp: at,
		ErrorType: "COMMUNICATION_ERROR", AttemptNumber: 1,
	}
	for _, tc := range []struct {
		name   string
		page   repo.ContractDeactivationDlqPage
		err    error
		mutate func(*repo.ContractDeactivationErrorEvent)
		want   int
	}{
		{name: "empty", want: 404},
		{name: "invalid token", err: repo.ErrInvalidDlqPagination, want: 400},
		{name: "database error", err: errors.New("secret database details"), want: 500},
		{name: "missing events", page: repo.ContractDeactivationDlqPage{
			Entries: []repo.ContractDeactivationDlqEntry{{ID: "wallet"}},
		}, want: 500},
		{name: "missing contract", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.ContractID = "" }, want: 500},
		{name: "invalid event ID", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.ID = "not-a-uuid" }, want: 500},
		{name: "invalid type", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.ErrorType = "invalid" }, want: 500},
		{name: "missing timestamp", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.Timestamp = time.Time{} }, want: 500},
		{name: "invalid attempt", mutate: func(e *repo.ContractDeactivationErrorEvent) { e.AttemptNumber = 0 }, want: 500},
		{name: "invalid HTTP code", mutate: func(e *repo.ContractDeactivationErrorEvent) {
			code := int32(600)
			e.HTTPErrorCode = &code
		}, want: 500},
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
			if recorder.Code != tc.want || !strings.Contains(recorder.Body.String(), `"status":`) {
				t.Errorf("want problem response %d, got %d: %s", tc.want, recorder.Code, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "secret database details") {
				t.Error("response exposed internal error")
			}
		})
	}
}
