package main

import (
	"context"
	. "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
)

type Server struct{}

var _ StrictServerInterface = (*Server)(nil)

func (*Server) GetServiceInfo(_ context.Context, _ GetServiceInfoRequestObject) (GetServiceInfoResponseObject, error) {
	return GetServiceInfo200JSONResponse{Version: "1.0.0"}, nil
}
