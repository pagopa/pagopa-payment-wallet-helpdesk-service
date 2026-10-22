package repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"pagopa.it/pagopa-payment-wallet-helpdesk-service/cmd/utils"
)

const (
	ContractDeactivationDlqCollectionNameEnvKey = "PAYMENT_WALLET_CONTRACT_DEACTIVATION_DLQ_COLLECTION_NAME"
	WalletLogEventsCollectionNameEnvKey         = "PAYMENT_WALLET_LOG_EVENTS_COLLECTION_NAME"
	ContractDeactivationErrorEventClass         = "it.pagopa.wallet.audit.WalletContractIdDeactivationErrorEvent"
)

var ErrInvalidDlqPagination = errors.New("invalid DLQ pagination")

type ContractDeactivationDlqEntry struct {
	ID        string `bson:"_id"`
	Timestamp string `bson:"timestamp"`
}

type ContractDeactivationErrorEvent struct {
	ID            string    `bson:"_id"`
	WalletID      string    `bson:"walletId"`
	ContractID    string    `bson:"contractId"`
	Timestamp     time.Time `bson:"timestamp"`
	ErrorType     string    `bson:"errorType"`
	HTTPErrorCode *int32    `bson:"httpErrorCode"`
	NPGErrorCode  *string   `bson:"npgErrorCode"`
	AttemptNumber int       `bson:"attemptNumber"`
	Malformed     bool      `bson:"-"`
}

func (entry *ContractDeactivationDlqEntry) UnmarshalBSON(data []byte) error {
	type document ContractDeactivationDlqEntry
	return decodeDlqTimestampDocument(data, (*document)(entry))
}

func (event *ContractDeactivationErrorEvent) UnmarshalBSON(data []byte) error {
	type document ContractDeactivationErrorEvent
	walletID, ok := bson.Raw(data).Lookup("walletId").StringValueOK()
	event.WalletID = walletID
	if !ok {
		event.Malformed = true
		return nil
	}
	if err := validateEventTimestamp(data); err != nil {
		event.Malformed = true
		return nil
	}
	var decoded document
	if err := bson.Unmarshal(data, &decoded); err != nil {
		event.Malformed = true
		return nil
	}
	*event = ContractDeactivationErrorEvent(decoded)
	return nil
}

func decodeDlqTimestampDocument(data []byte, target interface{}) error {
	if err := validateStringTimestamp(data); err != nil {
		return err
	}
	return bson.Unmarshal(data, target)
}

func validateStringTimestamp(data []byte) error {
	_, ok := bson.Raw(data).Lookup("timestamp").StringValueOK()
	if !ok {
		return errors.New("contract deactivation timestamp must be a string")
	}
	return nil
}

func validateEventTimestamp(data []byte) error {
	timestamp, ok := bson.Raw(data).Lookup("timestamp").StringValueOK()
	if !ok {
		return errors.New("contract deactivation timestamp must be an RFC3339 string")
	}
	if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
		return fmt.Errorf("invalid RFC3339 timestamp: %w", err)
	}
	return nil
}

type ContractDeactivationDlqPage struct {
	Entries   []ContractDeactivationDlqEntry
	Events    []ContractDeactivationErrorEvent
	NextToken string
}

type ContractDeactivationDlqRepository struct {
	dlq    *mongo.Collection
	events *mongo.Collection
}

func NewContractDeactivationDlqRepository(client *mongo.Client) *ContractDeactivationDlqRepository {
	db := client.Database(utils.GetEnvVariableOrDefault(PaymentWalletDBNameEnvKey, "wallet"))
	return &ContractDeactivationDlqRepository{
		dlq: db.Collection(utils.GetEnvVariableOrDefault(
			ContractDeactivationDlqCollectionNameEnvKey, "wallet-contractId-deactivation-dlq")),
		events: db.Collection(utils.GetEnvVariableOrDefault(
			WalletLogEventsCollectionNameEnvKey, "payment-wallets-log-events")),
	}
}

type dlqSequenceToken struct {
	Version   int    `json:"v"`
	Timestamp string `json:"t"`
	ID        string `json:"id"`
}

func decodeDlqSequenceToken(encoded string) (dlqSequenceToken, error) {
	var token dlqSequenceToken
	if len(encoded) > 512 {
		return token, ErrInvalidDlqPagination
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil {
		return token, ErrInvalidDlqPagination
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&token); err != nil {
		return token, ErrInvalidDlqPagination
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return token, ErrInvalidDlqPagination
	}
	var requiredFields struct {
		Timestamp *string `json:"t"`
		ID        *string `json:"id"`
	}
	if err := json.Unmarshal(data, &requiredFields); err != nil || requiredFields.Timestamp == nil || requiredFields.ID == nil {
		return token, ErrInvalidDlqPagination
	}
	if token.Version != 3 {
		return token, ErrInvalidDlqPagination
	}
	return token, nil
}

func (r *ContractDeactivationDlqRepository) Search(ctx context.Context, pageSize int, encodedToken string) (ContractDeactivationDlqPage, error) {
	var page ContractDeactivationDlqPage
	if pageSize < 1 || pageSize > 10 {
		return page, ErrInvalidDlqPagination
	}
	filter := bson.D{}
	if encodedToken != "" {
		token, err := decodeDlqSequenceToken(encodedToken)
		if err != nil {
			return page, err
		}
		at := token.Timestamp
		filter = bson.D{{Key: "$or", Value: bson.A{
			bson.D{{Key: "timestamp", Value: bson.D{{Key: "$lt", Value: at}}}},
			bson.D{
				{Key: "timestamp", Value: at},
				{Key: "_id", Value: bson.D{{Key: "$lt", Value: token.ID}}},
			},
		}}}
	}
	cursor, err := r.dlq.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(pageSize+1)).
		SetProjection(bson.D{{Key: "_id", Value: 1}, {Key: "timestamp", Value: 1}}))
	if err != nil {
		return page, fmt.Errorf("querying contract deactivation DLQ: %w", err)
	}
	if err := cursor.All(ctx, &page.Entries); err != nil {
		return page, fmt.Errorf("decoding contract deactivation DLQ: %w", err)
	}
	hasMore := len(page.Entries) > pageSize
	if hasMore {
		page.Entries = page.Entries[:pageSize]
	}
	if len(page.Entries) == 0 {
		return page, nil
	}
	ids := make([]string, len(page.Entries))
	for i, entry := range page.Entries {
		ids[i] = entry.ID
	}
	if hasMore {
		last := page.Entries[len(page.Entries)-1]
		data, err := json.Marshal(dlqSequenceToken{Version: 3, Timestamp: last.Timestamp, ID: last.ID})
		if err != nil {
			return page, fmt.Errorf("encoding DLQ continuation token: %w", err)
		}
		page.NextToken = base64.RawURLEncoding.EncodeToString(data)
	}
	eventFilter := bson.D{
		{Key: "walletId", Value: bson.D{{Key: "$in", Value: ids}}},
		{Key: "_class", Value: ContractDeactivationErrorEventClass},
	}
	cursor, err = r.events.Find(ctx, eventFilter, options.Find().SetProjection(bson.D{
		{Key: "_id", Value: 1}, {Key: "walletId", Value: 1}, {Key: "contractId", Value: 1},
		{Key: "timestamp", Value: 1}, {Key: "errorType", Value: 1}, {Key: "httpErrorCode", Value: 1},
		{Key: "npgErrorCode", Value: 1}, {Key: "attemptNumber", Value: 1},
	}))
	if err != nil {
		return page, fmt.Errorf("querying contract deactivation error events: %w", err)
	}
	if err := cursor.All(ctx, &page.Events); err != nil {
		return page, fmt.Errorf("decoding contract deactivation error events: %w", err)
	}
	return page, nil
}
