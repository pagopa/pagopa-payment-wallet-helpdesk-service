package main

import (
	"log"
	"path/filepath"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"

	"net/http"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	DB "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/cosmosdb"
	service "pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/service"
)

func main() {
	cosmosdb, _ := DB.GetMongoClient()
	defer DB.CloseMongoClient()
	validatorPath, err := filepath.Abs("./api-spec/api.yaml")
	if err != nil {
		log.Fatal("Unable to get path to api spec!")
	}

	validator, err := ginmiddleware.OapiValidatorFromYamlFile(validatorPath)
	if err != nil {
		log.Fatalf("Unable to get api spec: unable to read validator in path %s", validatorPath)
	}
	paymentWalletHelpdeskService := service.NewPaymentWalletHelpdeskService(cosmosdb)
	server := api.NewStrictHandler(&Server{
		paymentWalletHelpdeskService: paymentWalletHelpdeskService,
	}, []api.StrictMiddlewareFunc{})
	r := gin.Default()

	r.Use(validator)

	api.RegisterHandlers(r, server)

	s := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8080",
	}

	log.Fatal(s.ListenAndServe())
}
