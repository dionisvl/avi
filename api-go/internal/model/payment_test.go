package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dionisvl/avi/api-go/internal/model"
)

func TestMoney_NewMoneyValidates(t *testing.T) {
	tests := []struct {
		name        string
		amountMinor int64
		currency    string
		wantErr     bool
	}{
		{"valid RUB", 10000, "RUB", false},
		{"negative amount", -100, "RUB", true},
		{"zero amount", 0, "RUB", true},
		{"unsupported currency", 10000, "USD", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := model.NewMoney(tt.amountMinor, tt.currency)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMoney_Format(t *testing.T) {
	m, _ := model.NewMoney(10000, "RUB")
	assert.Equal(t, "100.00", m.Format())

	m, _ = model.NewMoney(1, "RUB")
	assert.Equal(t, "0.01", m.Format())

	m, _ = model.NewMoney(12345, "RUB")
	assert.Equal(t, "123.45", m.Format())
}

func TestPayment_NewPaymentCreatesValidPending(t *testing.T) {
	userID := uuid.New()
	subjectID := uuid.New()
	amount, _ := model.NewMoney(10000, "RUB")

	payment := model.NewPayment(userID, model.PaymentPurposePromoteListing, subjectID, amount)

	assert.Equal(t, userID, payment.UserID)
	assert.Equal(t, model.PaymentPurposePromoteListing, payment.Purpose)
	assert.Equal(t, subjectID, payment.SubjectID)
	assert.Equal(t, model.PaymentStatusPending, payment.Status)
	assert.Equal(t, model.PaymentProviderYooKassa, payment.Provider)
	assert.NotZero(t, payment.ID)
	assert.Equal(t, payment.ID.String(), payment.IdempotencyKey)
	assert.Nil(t, payment.PaidAt)
	assert.Nil(t, payment.CanceledAt)
}

func TestPayment_CanApplyProviderStatus(t *testing.T) {
	userID := uuid.New()
	subjectID := uuid.New()
	amount, _ := model.NewMoney(10000, "RUB")
	payment := model.NewPayment(userID, model.PaymentPurposePromoteListing, subjectID, amount)

	// pending can go to succeeded
	assert.True(t, payment.CanApplyProviderStatus(model.PaymentStatusSucceeded))

	// pending can go to canceled
	assert.True(t, payment.CanApplyProviderStatus(model.PaymentStatusCanceled))

	// once succeeded, cannot go to canceled
	payment.Status = model.PaymentStatusSucceeded
	assert.False(t, payment.CanApplyProviderStatus(model.PaymentStatusCanceled))

	// once succeeded, can stay succeeded
	assert.True(t, payment.CanApplyProviderStatus(model.PaymentStatusSucceeded))
}

func TestPaymentPurpose_Valid(t *testing.T) {
	assert.True(t, model.PaymentPurposePromoteListing.Valid())
	assert.True(t, model.PaymentPurposeListingPlacement.Valid())
	assert.True(t, model.PaymentPurposeListingBoost.Valid())
	assert.True(t, model.PaymentPurposeSubscription.Valid())

	assert.False(t, model.PaymentPurpose("unknown").Valid())
}
