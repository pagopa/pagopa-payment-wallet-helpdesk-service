package repository

var PaymentWalletDiscriminatorMap = map[string]string{
	"CARDS":  "it.pagopa.wallet.documents.wallets.details.CardDetails",
	"PAYPAL": "it.pagopa.wallet.documents.wallets.details.PayPalDetails",
}

var reverseLookupMap = func() map[string]string {
	mapping := make(map[string]string)
	for k, v := range PaymentWalletDiscriminatorMap {
		mapping[v] = k
	}
	return mapping
}

// WalletDetailsModel wallet details model: only the needed fields are extracted from DB
type WalletDetailsModel struct {
	ClassDiscriminatorField string `json:"_class" bson:"_class"`
}

// ApplicationModel application model: only the needed fields are extracted from DB
type ApplicationModel struct {
	ID     string `json:"_id" bson:"_id"`
	Status string `json:"status" bson:"status"`
}

// WalletModel wallet structure interface: only the needed fields are extracted from DB
type WalletModel struct {
	ID                string             `json:"_id" bson:"_id"`
	UserID            string             `json:"userId" bson:"userId"`
	Status            string             `json:"status" bson:"status"`
	OnboardingChannel string             `json:"onboardingChannel" bson:"onboardingChannel"`
	Details           WalletDetailsModel `json:"details" bson:"details"`
	Applications      []ApplicationModel `json:"applications" bson:"applications"`
}

func (model *WalletDetailsModel) GetDetailType() string {
	return reverseLookupMap()[model.ClassDiscriminatorField]
}
