package main

import (
	"log"
	"path/filepath"

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

	validator, err := newRequestValidator(validatorPath)
	if err != nil {
		log.Fatalf("Unable to load API validator: %v", err)
	}
	paymentWalletHelpdeskService := service.NewPaymentWalletHelpdeskService(cosmosdb)
	server := newServerHandler(&Server{
		paymentWalletHelpdeskService: paymentWalletHelpdeskService,
	})
	r := gin.Default()

	r.Use(validator)

	api.RegisterHandlers(r, server)

	s := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8080",
	}

	log.Fatal(s.ListenAndServe())
}
