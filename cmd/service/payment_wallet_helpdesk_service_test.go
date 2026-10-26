package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	repo "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/repository"
)

type paymentWalletRepositoryStub struct {
	reset func(context.Context, string) (bool, error)
	get   func(string, *string, *string, context.Context) ([]repo.WalletModel, error)
}

func (r paymentWalletRepositoryStub) ResetDeactivationRetryCount(ctx context.Context, id string) (bool, error) {
	return r.reset(ctx, id)
}

func (r paymentWalletRepositoryStub) GetWallets(userID string, status, walletType *string, ctx context.Context) ([]repo.WalletModel, error) {
	return r.get(userID, status, walletType, ctx)
}

func TestResetDeactivationRetryCountMapsRepositoryResults(t *testing.T) {
	for _, tc := range []struct {
		name  string
		found bool
		err   error
		want  int
	}{
		{name: "existing wallet", found: true, want: http.StatusNoContent},
		{name: "missing wallet", want: http.StatusNotFound},
		{name: "database failure", err: errors.New("secret database details"), want: http.StatusInternalServerError},
		{name: "error takes precedence over found", found: true, err: errors.New("secret database details"), want: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			originalOutput := log.Writer()
			log.SetOutput(&logs)
			t.Cleanup(func() { log.SetOutput(originalOutput) })
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			const walletID = "00000000-0000-4000-8000-000000000001"
			calls := 0
			s := &PaymentWalletHelpdeskService{paymentWalletRepository: paymentWalletRepositoryStub{
				reset: func(gotCtx context.Context, gotID string) (bool, error) {
					calls++
					if gotCtx != ctx || gotID != walletID {
						t.Fatalf("wrong reset arguments: ctx=%v, walletID=%q", gotCtx, gotID)
					}
					return tc.found, tc.err
				},
			}}
			response, err := s.ResetDeactivationRetryCount(ctx, api.ResetWalletDeactivationRetryCountRequestObject{
				WalletId: uuid.MustParse(walletID),
			})
			if err != nil {
				t.Fatalf("expected typed response, got error: %v", err)
			}
			if calls != 1 {
				t.Fatalf("reset calls = %d, want 1", calls)
			}
			recorder := httptest.NewRecorder()
			if err := response.VisitResetWalletDeactivationRetryCountResponse(recorder); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, tc.want, recorder.Body.String())
			}
			if tc.want == http.StatusNoContent {
				if recorder.Body.Len() != 0 {
					t.Fatalf("expected empty response, got %s", recorder.Body.String())
				}
				if !strings.Contains(logs.String(), walletID) || !strings.Contains(logs.String(), "Reset completed") {
					t.Fatalf("missing successful reset log with wallet ID: %q", logs.String())
				}
				return
			}
			if strings.Contains(logs.String(), "Reset completed") {
				t.Fatalf("failed reset logged as successful: %s", logs.String())
			}
			var problem api.ProblemJson
			if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil ||
				problem.Status == nil || int(*problem.Status) != tc.want ||
				problem.Title == nil || *problem.Title == "" || problem.Detail == nil || *problem.Detail == "" {
				t.Fatalf("expected typed %d problem, got %s", tc.want, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "secret database details") {
				t.Fatalf("internal details leaked: %s", recorder.Body.String())
			}
			if tc.err != nil && !strings.Contains(logs.String(), tc.err.Error()) {
				t.Fatalf("database error was not logged: %s", logs.String())
			}
		})
	}
}

func TestSearchWalletsPreservesRepositoryFiltersAndResponse(t *testing.T) {
	ctx := context.Background()
	status := api.WalletStatus("VALIDATED")
	walletType := api.WalletType("CARDS")
	s := &PaymentWalletHelpdeskService{paymentWalletRepository: paymentWalletRepositoryStub{
		get: func(userID string, gotStatus, gotType *string, gotCtx context.Context) ([]repo.WalletModel, error) {
			if userID != "user" || gotStatus == nil || *gotStatus != "VALIDATED" ||
				gotType == nil || *gotType != "CARDS" || gotCtx != ctx {
				t.Fatal("wallet search filters or context were not forwarded")
			}
			return []repo.WalletModel{{
				ID: "wallet", Status: "VALIDATED", OnboardingChannel: "IO",
				Details:      repo.WalletDetailsModel{ClassDiscriminatorField: "it.pagopa.wallet.documents.wallets.details.CardDetails"},
				Applications: []repo.ApplicationModel{{ID: "IO", Status: "ENABLED"}},
			}}, nil
		},
	}}
	response, err := s.SearchWallets(ctx, api.GetWalletsRequestObject{
		UserId: "user", Params: api.GetWalletsParams{Status: &status, Type: &walletType},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	if err := response.VisitGetWalletsResponse(recorder); err != nil {
		t.Fatal(err)
	}
	var body api.WalletsDetails
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || len(body.Wallets) != 1 {
		t.Fatalf("unexpected wallet response: %d %s", recorder.Code, recorder.Body.String())
	}
	wallet := body.Wallets[0]
	if wallet.Id != "wallet" || wallet.Status != "VALIDATED" || wallet.Type != "CARDS" ||
		wallet.OnboardingChannel != "IO" || len(wallet.Applications) != 1 ||
		wallet.Applications[0].Name != "IO" || wallet.Applications[0].Status != "ENABLED" {
		t.Fatalf("wallet details were not preserved: %+v", wallet)
	}
}
