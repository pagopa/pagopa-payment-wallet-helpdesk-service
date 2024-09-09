package repository

// WalletDetailsModel wallet details model: only the needed fields are extracted from DB
type WalletDetailsModel struct {
	Type string `json:"type" bson:"type"`
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
