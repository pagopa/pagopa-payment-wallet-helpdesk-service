package main

import (
	"log"
	"path/filepath"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"net/http"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	DB "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/cosmosdb"
	service "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/service"
)

func main() {
	cosmosdb, err := DB.GetMongoClient()
	if err != nil {
		log.Fatalf("Unable to get mongo client: %v", err)
	}
	defer DB.CloseMongoClient()
	validatorPath, err := filepath.Abs("./api-spec/api.yaml")
	if err != nil {
		log.Fatalf("Unable to get path to api spec!: %v", err)
	}

	paymentWalletHelpdeskService := service.NewPaymentWalletHelpdeskService(cosmosdb)
	r, err := newRouter(&Server{
		paymentWalletHelpdeskService:   paymentWalletHelpdeskService,
		contractDeactivationDlqService: service.NewContractDeactivationDlqService(cosmosdb),
	}, validatorPath)
	if err != nil {
		log.Fatalf("Unable to configure HTTP router: %v", err)
	}

	s := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8080",
	}

	log.Fatal(s.ListenAndServe())
}

func newRouter(server api.StrictServerInterface, validatorPath string) (*gin.Engine, error) {
	apiKeyFilter, err := newAPIKeyFilter()
	if err != nil {
		return nil, err
	}
	validator, err := newRequestValidator(validatorPath)
	if err != nil {
		return nil, err
	}
	router := gin.New()
	router.Use(gin.Logger(), gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		// Gin's default recovery dumps request headers, including x-api-key.
		log.Printf("Recovered HTTP handler panic\n%s", debug.Stack())
		c.AbortWithStatus(http.StatusInternalServerError)
	}))
	router.Use(apiKeyFilter, validator)
	api.RegisterHandlers(router, newServerHandler(server))
	return router, nil
}
