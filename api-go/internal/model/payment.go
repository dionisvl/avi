package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PaymentProvider string

const (
	PaymentProviderYooKassa PaymentProvider = "yookassa"
)

type PaymentPurpose string

const (
	PaymentPurposePromoteListing   PaymentPurpose = "promote_listing"
	PaymentPurposeDemoCheckout     PaymentPurpose = "demo_checkout"
	PaymentPurposeListingPlacement PaymentPurpose = "listing_placement"
	PaymentPurposeListingBoost     PaymentPurpose = "listing_boost"
	PaymentPurposeSubscription     PaymentPurpose = "subscription"
)

func (p PaymentPurpose) String() string {
	return string(p)
}

func (p PaymentPurpose) Valid() bool {
	switch p {
	case PaymentPurposePromoteListing, PaymentPurposeDemoCheckout, PaymentPurposeListingPlacement,
		PaymentPurposeListingBoost, PaymentPurposeSubscription:
		return true
	}
	return false
}

type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusSucceeded         PaymentStatus = "succeeded"
	PaymentStatusCanceled          PaymentStatus = "canceled"
	PaymentStatusRefunded          PaymentStatus = "refunded"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
)

func (s PaymentStatus) String() string {
	return string(s)
}

type Money struct {
	amountMinor int64
	currency    string
}

func NewMoney(amountMinor int64, currency string) (Money, error) {
	if amountMinor <= 0 {
		return Money{}, fmt.Errorf("amount must be positive, got %d", amountMinor)
	}
	if currency != "RUB" {
		return Money{}, fmt.Errorf("unsupported currency: %s", currency)
	}
	return Money{
		amountMinor: amountMinor,
		currency:    currency,
	}, nil
}

func (m Money) AmountMinor() int64 {
	return m.amountMinor
}

func (m Money) Currency() string {
	return m.currency
}

func (m Money) IsPositive() bool {
	return m.amountMinor > 0
}

// Format returns the amount formatted as "XXX.XX" for API response
func (m Money) Format() string {
	return fmt.Sprintf("%d.%02d", m.amountMinor/100, m.amountMinor%100)
}

type Payment struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Purpose           PaymentPurpose
	SubjectID         uuid.UUID
	Amount            Money
	Status            PaymentStatus
	Provider          PaymentProvider
	ProviderPaymentID string
	ConfirmationURL   string
	IdempotencyKey    string
	ProviderMethodID  *string
	PaidAt            *time.Time
	CanceledAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewPayment(
	userID uuid.UUID,
	purpose PaymentPurpose,
	subjectID uuid.UUID,
	amount Money,
) *Payment {
	now := time.Now()
	id := uuid.New()
	return &Payment{
		ID:             id,
		UserID:         userID,
		Purpose:        purpose,
		SubjectID:      subjectID,
		Amount:         amount,
		Status:         PaymentStatusPending,
		Provider:       PaymentProviderYooKassa,
		IdempotencyKey: id.String(),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (p *Payment) MarkSucceeded(providerPaymentID string, metadata map[string]any) {
	now := time.Now()
	p.Status = PaymentStatusSucceeded
	p.ProviderPaymentID = providerPaymentID
	p.PaidAt = &now
	p.UpdatedAt = now
}

func (p *Payment) MarkCanceled() {
	now := time.Now()
	p.Status = PaymentStatusCanceled
	p.CanceledAt = &now
	p.UpdatedAt = now
}

func (p *Payment) CanApplyProviderStatus(newStatus PaymentStatus) bool {
	// cannot flip a succeeded payment to canceled
	if p.Status == PaymentStatusSucceeded && newStatus == PaymentStatusCanceled {
		return false
	}
	return true
}
