package main

import (
	"context"
	_ "embed"
	. "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
)

//go:embed version.txt
var version string

type Server struct{}

var _ StrictServerInterface = (*Server)(nil)

func (*Server) GetServiceInfo(_ context.Context, _ GetServiceInfoRequestObject) (GetServiceInfoResponseObject, error) {
	return GetServiceInfo200JSONResponse{Version: version}, nil
}
