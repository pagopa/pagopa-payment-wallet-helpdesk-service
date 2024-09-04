package main

import (
	"log"
	"syscall"

	PaymentWalletRepository "pagopa.it/pagopa-payment-wallet-helpdesk-service/internal/repository"
)

func main() {
	wallets, err := PaymentWalletRepository.GetWalletsByUserID("00000000-0000-0000-0000-000000000000")
	if err != nil {
		log.Printf("Error searching for wallets %v", err)
		syscall.Exit(1)
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
		syscall.Exit(0)
	}
}
