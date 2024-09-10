package service

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	api "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	repo "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/repository"
)

type PaymentWalletHelpdeskService struct {
	paymentWalletRepository repo.PaymentWalletRepository
}

func NewPaymentWalletHelpdeskService(db *mongo.Client) *PaymentWalletHelpdeskService {
	return &PaymentWalletHelpdeskService{
		paymentWalletRepository: *repo.NewPaymentWalletRepository(db),
	}
}

func (p *PaymentWalletHelpdeskService) SearchWallets(c context.Context, request api.GetWalletsRequestObject) (api.GetWalletsResponseObject, error) {
	userID := request.UserId
	walletStatus := request.Params.Status
	walletType := request.Params.Type
	log.Printf("Searching wallet for input request parameters: %v", request)
	wallets, err := p.paymentWalletRepository.GetWallets(userID, (*string)(walletStatus), (*string)(walletType), c)
	if err != nil {
		log.Printf("error searching for wallets: [%v]", err)
		return api.GetWallets500JSONResponse(p.buildProblemJSON(500, fmt.Sprintf("%v", err))), err

	}
	if len(wallets) == 0 {
		log.Print("No wallet found for input searching parameters")
		return api.GetWallets404JSONResponse(p.buildProblemJSON(404, "No wallet found")), err
	}
	var walletDetailsDto []api.WalletDetail
	var wallet repo.WalletModel
	var application repo.ApplicationModel
	for _, wallet = range wallets {
		var applications []api.WalletApplicationInfo
		for _, application = range wallet.Applications {
			applications = append(applications, api.WalletApplicationInfo{
				Name:   application.ID,
				Status: api.WalletApplicationStatus(application.Status),
			})
		}
		walletDetailsDto = append(walletDetailsDto, api.WalletDetail{
			Id:                wallet.ID,
			OnboardingChannel: wallet.OnboardingChannel,
			Status:            wallet.Status,
			Type:              api.WalletType(wallet.Details.GetDetailType()),
			Applications:      applications,
		})
	}
	return api.GetWallets200JSONResponse{
		Wallets: walletDetailsDto,
	}, err
}

func (p *PaymentWalletHelpdeskService) buildProblemJSON(httpErrorCode int32, errorCause string) api.ProblemJson {
	detail := fmt.Sprintf("error: [%v]", errorCause)
	errorTitle := "Error searching wallets"
	return api.ProblemJson{
		Detail: &detail,
		Status: &httpErrorCode,
		Title:  &errorTitle,
	}
}
