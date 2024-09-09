package main

import (
	"context"
	_ "embed"

	api "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	service "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/service"
)

//go:embed version.txt
var version string

type Server struct {
	paymentWalletHelpdeskService *service.PaymentWalletHelpdeskService
}

var _ api.StrictServerInterface = (*Server)(nil)

func (*Server) GetServiceInfo(_ context.Context, _ api.GetServiceInfoRequestObject) (api.GetServiceInfoResponseObject, error) {
	return api.GetServiceInfo200JSONResponse{Version: version}, nil
}

func (s *Server) GetWallets(c context.Context, params api.GetWalletsRequestObject) (api.GetWalletsResponseObject, error) {
	return s.paymentWalletHelpdeskService.SearchWallets(c, params)
}
