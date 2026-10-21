package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	repo "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/repository"
)

type dlqRepository interface {
	Search(context.Context, int, string) (repo.ContractDeactivationDlqPage, error)
}

type ContractDeactivationDlqService struct {
	repository dlqRepository
}

func NewContractDeactivationDlqService(client *mongo.Client) *ContractDeactivationDlqService {
	return &ContractDeactivationDlqService{repository: repo.NewContractDeactivationDlqRepository(client)}
}

func (s *ContractDeactivationDlqService) Search(ctx context.Context, request api.GetWalletContractIdDeactivationDlqRequestObject) (api.GetWalletContractIdDeactivationDlqResponseObject, error) {
	token := ""
	if request.Params.SearchSequenceToken != nil {
		token = *request.Params.SearchSequenceToken
	}
	page, err := s.repository.Search(ctx, request.Params.PageSize, token)
	if errors.Is(err, repo.ErrInvalidDlqPagination) {
		return api.GetWalletContractIdDeactivationDlq400JSONResponse(dlqProblem(400, "Invalid page size or continuation token")), nil
	}
	if err != nil {
		log.Printf("Error searching contract deactivation DLQ: %v", err)
		return api.GetWalletContractIdDeactivationDlq500JSONResponse(dlqProblem(500, "Unable to retrieve contract deactivation failures")), nil
	}
	if len(page.Entries) == 0 {
		return api.GetWalletContractIdDeactivationDlq404JSONResponse(dlqProblem(404, "No DLQ entries found")), nil
	}
	eventsByWallet := make(map[string][]repo.ContractDeactivationErrorEvent)
	for _, event := range page.Events {
		eventsByWallet[event.WalletID] = append(eventsByWallet[event.WalletID], event)
	}
	results := make([]api.WalletContractIdDeactivationDlqDetail, 0, len(page.Entries))
	for _, entry := range page.Entries {
		detail, err := dlqDetail(entry.ID, eventsByWallet[entry.ID])
		if err != nil {
			log.Printf("Inconsistent contract deactivation DLQ data: %v", err)
			return api.GetWalletContractIdDeactivationDlq500JSONResponse(dlqProblem(500, "Inconsistent contract deactivation failure data")), nil
		}
		results = append(results, detail)
	}
	return api.GetWalletContractIdDeactivationDlq200JSONResponse{
		Results: results, SearchSequenceToken: page.NextToken,
	}, nil
}

func dlqProblem(status int32, detail string) api.ProblemJson {
	title := "Contract ID deactivation DLQ search"
	return api.ProblemJson{Status: &status, Title: &title, Detail: &detail}
}

func dlqDetail(walletID string, events []repo.ContractDeactivationErrorEvent) (api.WalletContractIdDeactivationDlqDetail, error) {
	result := api.WalletContractIdDeactivationDlqDetail{Id: walletID}
	if len(events) == 0 {
		return result, fmt.Errorf("wallet %s has no deactivation error events", walletID)
	}
	latest := events[0]
	for _, event := range events {
		errorType := api.WalletContractIdDeactivationErrorErrorType(event.ErrorType)
		id, err := uuid.Parse(event.ID)
		if err != nil || id.String() != event.ID || event.ContractID == "" || event.Timestamp.IsZero() || event.AttemptNumber < 1 ||
			!errorType.Valid() || (event.HTTPErrorCode != nil && (*event.HTTPErrorCode < 100 || *event.HTTPErrorCode >= 600)) {
			return result, fmt.Errorf("wallet %s has a malformed deactivation error event", walletID)
		}
		if event.Timestamp.After(latest.Timestamp) || (event.Timestamp.Equal(latest.Timestamp) &&
			(event.AttemptNumber > latest.AttemptNumber || (event.AttemptNumber == latest.AttemptNumber && event.ID > latest.ID))) {
			latest = event
		}
	}
	result.ContractId = latest.ContractID
	sort.Slice(events, func(i, j int) bool {
		if events[i].AttemptNumber != events[j].AttemptNumber {
			return events[i].AttemptNumber < events[j].AttemptNumber
		}
		if !events[i].Timestamp.Equal(events[j].Timestamp) {
			return events[i].Timestamp.Before(events[j].Timestamp)
		}
		return events[i].ID < events[j].ID
	})
	for _, event := range events {
		if event.ContractID == result.ContractId {
			result.DeactivationErrors = append(result.DeactivationErrors, api.WalletContractIdDeactivationError{
				ErrorType:     api.WalletContractIdDeactivationErrorErrorType(event.ErrorType),
				HttpErrorCode: event.HTTPErrorCode, NpgErrorCode: event.NPGErrorCode,
			})
		}
	}
	return result, nil
}
