package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/gin-middleware"
	"log"
	"path/filepath"
	"time"

	"net/http"
	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/app/api"
	"pagopa.it/pagopa-payment-wallet-helpdesk-service/internal/cosmosdb"
	"pagopa.it/pagopa-payment-wallet-helpdesk-service/internal/repository"
)

func main() {
	mongoClient, err := cosmosdb.GetMongoClient()
	if err != nil {
		log.Panicf("Error connecting to MongoDB %v", err)
	}
	defer cosmosdb.CloseMongoClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	paymentWalletRepository := repository.NewPaymentWalletRepository(mongoClient)
	wallets, err := paymentWalletRepository.GetWalletsByUserID("6fd46190-f4bf-4a9f-9e70-fb98f235a449", ctx)
	if err != nil {
		log.Printf("Error searching for wallets %v", err)
	} else {
		totalWallets := len(wallets)
		for idx, wallet := range wallets {
			log.Printf("Wallet %d/%d -> %s", idx+1, totalWallets, wallet)
			log.Printf("wallet id: [%s]", wallet.ID)
			log.Printf("onboarding channel: [%s]", wallet.OnboardingChannel)
			log.Printf("wallet status: [%s]", wallet.Status)
			log.Printf("wallet user id: [%s]", wallet.UserID)
			log.Printf("wallet detail type: [%s]", wallet.Details.Type)
			if len(wallet.Applications) > 0 {
				log.Printf("wallet application ID: [%s]", wallet.Applications[0].ID)
				log.Printf("wallet application status: [%s]", wallet.Applications[0].Status)
			}
		}
	}

	validatorPath, err := filepath.Abs("./api-spec/api.yaml")
	if err != nil {
		log.Fatal("Unable to get path to api spec!")
	}

	validator, err := ginmiddleware.OapiValidatorFromYamlFile(validatorPath)
	if err != nil {
		log.Fatalf("Unable to get api spec: unable to read validator in path %s", validatorPath)
	}

	server := api.NewStrictHandler(&Server{}, []api.StrictMiddlewareFunc{})
	r := gin.Default()

	r.Use(validator)

	api.RegisterHandlers(r, server)

	s := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8080",
	}

	log.Fatal(s.ListenAndServe())
}
