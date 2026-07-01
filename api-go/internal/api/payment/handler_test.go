package payment

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dionisvl/avi/api-go/internal/config"
	"github.com/dionisvl/avi/api-go/internal/model"
	paymentservice "github.com/dionisvl/avi/api-go/internal/service/payment"
)

type fakeService struct {
	webhookErr error
}

func (f *fakeService) CreatePayment(ctx context.Context, in paymentservice.CreatePaymentInput) (*paymentservice.CreatePaymentResult, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeService) HandleProviderEvent(ctx context.Context, provider model.PaymentProvider, payload []byte) error {
	return f.webhookErr
}

func (f *fakeService) HasEntitlement(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (bool, error) {
	return false, nil
}

func TestWebhookYookassa_Returns500OnProcessingError(t *testing.T) {
	svc := &fakeService{webhookErr: errors.New("database unavailable")}
	handler := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhooks/yookassa", http.NoBody)
	rr := httptest.NewRecorder()

	handler.webhookYookassa(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestWebhookYookassa_Returns200OnIgnoredEvent(t *testing.T) {
	svc := &fakeService{}
	handler := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhooks/yookassa", http.NoBody)
	rr := httptest.NewRecorder()

	handler.webhookYookassa(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func newTestHandler(svc paymentservice.Service) *Handler {
	cfg := config.Load()
	return NewHandler(svc, cfg.Payments.ReturnURL, cfg.SMTP.FrontendDomain, slog.Default())
}
