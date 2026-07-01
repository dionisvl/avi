package provider

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/model"
)

type Provider interface {
	CreatePayment(ctx context.Context, in CreatePaymentInput) (*CreatedPayment, error)
	GetPayment(ctx context.Context, providerPaymentID string) (*PaymentInfo, error)
	ParseWebhookEvent(payload []byte) (*WebhookEvent, error)
}

type CreatePaymentInput struct {
	LocalPaymentID uuid.UUID
	UserID         uuid.UUID
	Purpose        model.PaymentPurpose
	SubjectID      uuid.UUID
	Amount         model.Money
	Description    string
	ReturnURL      string
	IdempotencyKey string
	BuyerEmail     string
	SaveMethod     bool // reserved for recurrent payments
}

type CreatedPayment struct {
	ProviderPaymentID string
	Status            model.PaymentStatus
	ConfirmationURL   string
	ProviderMethodID  *string
	Metadata          map[string]any
	// Receipt is the raw fiscal receipt sent to the provider, stored locally
	// for audit. Nil when no receipt was sent.
	Receipt json.RawMessage
}

type PaymentInfo struct {
	ProviderPaymentID string
	Status            model.PaymentStatus
	ProviderStatus    string
	Metadata          map[string]any
}

type WebhookEvent struct {
	ProviderPaymentID string
	EventType         string
	ProviderStatus    string
	EventKey          string
}
