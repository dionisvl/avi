package yookassa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/payment/provider"
)

const BaseURL = "https://api.yookassa.ru/v3/"

type ReceiptConfig struct {
	VatCode        int
	PaymentSubject string
	PaymentMode    string
}

type Adapter struct {
	shopID     string
	secretKey  string
	receiptCfg ReceiptConfig
	httpClient *http.Client
}

func NewAdapter(shopID, secretKey string, receiptCfg ReceiptConfig) *Adapter {
	return &Adapter{
		shopID:     shopID,
		secretKey:  secretKey,
		receiptCfg: receiptCfg,
		httpClient: &http.Client{},
	}
}

type receiptCustomer struct {
	Email string `json:"email"`
}

type receiptItem struct {
	Description    string            `json:"description"`
	Quantity       string            `json:"quantity"`
	Amount         receiptItemAmount `json:"amount"`
	VatCode        int               `json:"vat_code"`
	PaymentSubject string            `json:"payment_subject"`
	PaymentMode    string            `json:"payment_mode"`
}

type receiptItemAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type receipt struct {
	Customer receiptCustomer `json:"customer"`
	Items    []receiptItem   `json:"items"`
}

type createPaymentRequest struct {
	Amount struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"amount"`
	Confirmation struct {
		Type      string `json:"type"`
		ReturnURL string `json:"return_url"`
	} `json:"confirmation"`
	Capture        bool              `json:"capture"`
	Description    string            `json:"description"`
	IdempotencyKey string            `json:"idempotency_key"`
	Metadata       map[string]string `json:"metadata"`
	Receipt        *receipt          `json:"receipt,omitempty"`
}

type createPaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		Type            string `json:"type"`
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

func (a *Adapter) CreatePayment(ctx context.Context, in provider.CreatePaymentInput) (*provider.CreatedPayment, error) {
	// Convert internal amount to YooKassa format
	amountStr := formatAmount(in.Amount)

	// Build description from purpose
	description := descriptionFromPurpose(in.Purpose)

	// Create request payload
	currency := in.Amount.Currency()
	req := createPaymentRequest{}
	req.Amount.Value = amountStr
	req.Amount.Currency = currency
	req.Confirmation.Type = "redirect"
	req.Confirmation.ReturnURL = in.ReturnURL
	req.Capture = true
	req.Description = description
	req.IdempotencyKey = in.IdempotencyKey
	if in.BuyerEmail != "" {
		req.Receipt = &receipt{
			Customer: receiptCustomer{Email: in.BuyerEmail},
			Items: []receiptItem{
				{
					Description:    description,
					Quantity:       "1.00",
					Amount:         receiptItemAmount{Value: amountStr, Currency: currency},
					VatCode:        a.receiptCfg.VatCode,
					PaymentSubject: a.receiptCfg.PaymentSubject,
					PaymentMode:    a.receiptCfg.PaymentMode,
				},
			},
		}
	}
	req.Metadata = map[string]string{
		"local_payment_id": in.LocalPaymentID.String(),
		"user_id":          in.UserID.String(),
		"subject_id":       in.SubjectID.String(),
	}

	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", BaseURL+"payments", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotence-Key", in.IdempotencyKey)
	httpReq.SetBasicAuth(a.shopID, a.secretKey)

	// Execute request
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment at provider: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("provider error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var payment createPaymentResponse
	if err := json.Unmarshal(respBody, &payment); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Map status
	status := mapProviderStatus(payment.Status)

	// Build metadata
	meta := map[string]any{
		"provider_status": payment.Status,
		"created_at":      payment.CreatedAt,
		"expires_at":      payment.ExpiresAt,
	}

	var receiptJSON json.RawMessage
	if req.Receipt != nil {
		if raw, err := json.Marshal(req.Receipt); err == nil {
			receiptJSON = raw
		}
	}

	return &provider.CreatedPayment{
		ProviderPaymentID: payment.ID,
		Status:            status,
		ConfirmationURL:   payment.Confirmation.ConfirmationURL,
		Metadata:          meta,
		Receipt:           receiptJSON,
	}, nil
}

func (a *Adapter) GetPayment(ctx context.Context, providerPaymentID string) (*provider.PaymentInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", BaseURL+"payments/"+url.PathEscape(providerPaymentID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment status request: %w", err)
	}

	httpReq.SetBasicAuth(a.shopID, a.secretKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment from provider: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read payment status response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("provider status error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var payment createPaymentResponse
	if err := json.Unmarshal(respBody, &payment); err != nil {
		return nil, fmt.Errorf("failed to parse payment status response: %w", err)
	}

	return &provider.PaymentInfo{
		ProviderPaymentID: payment.ID,
		Status:            mapProviderStatus(payment.Status),
		ProviderStatus:    payment.Status,
		Metadata: map[string]any{
			"provider_status": payment.Status,
			"created_at":      payment.CreatedAt,
			"expires_at":      payment.ExpiresAt,
		},
	}, nil
}

func formatAmount(m model.Money) string {
	// YooKassa expects decimal format: 10000 minor units -> "100.00"
	return m.Format()
}

func descriptionFromPurpose(purpose model.PaymentPurpose) string {
	switch purpose {
	case model.PaymentPurposePromoteListing:
		return "Listing promotion"
	case model.PaymentPurposeListingPlacement:
		return "Listing placement"
	case model.PaymentPurposeListingBoost:
		return "Listing boost"
	case model.PaymentPurposeSubscription:
		return "Subscription"
	default:
		return "Payment"
	}
}

func mapProviderStatus(status string) model.PaymentStatus {
	switch status {
	case "pending", "waiting_for_capture":
		return model.PaymentStatusPending
	case "succeeded":
		return model.PaymentStatusSucceeded
	case "canceled":
		return model.PaymentStatusCanceled
	default:
		return model.PaymentStatusPending
	}
}

func (a *Adapter) ParseWebhookEvent(payload []byte) (*provider.WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}
	if event.Type != "notification" {
		return nil, fmt.Errorf("invalid webhook type: %s", event.Type)
	}
	if event.Event == "" {
		return nil, fmt.Errorf("missing webhook event")
	}
	if event.Object.ID == "" {
		return nil, fmt.Errorf("missing webhook object id")
	}
	if event.Object.Status == "" {
		return nil, fmt.Errorf("missing webhook object status")
	}
	return &provider.WebhookEvent{
		ProviderPaymentID: event.Object.ID,
		EventType:         event.Event,
		ProviderStatus:    event.Object.Status,
		EventKey:          event.eventKey(),
	}, nil
}

// WebhookEvent represents a YooKassa notification.
// Format per https://yookassa.ru/developers/using-api/webhooks:
// {"type": "notification", "event": "payment.succeeded", "object": {"id": "...", "status": "succeeded", ...}}
type WebhookEvent struct {
	Type   string `json:"type"`  // always "notification"
	Event  string `json:"event"` // e.g. "payment.succeeded"
	Object struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"object"`
}

func (w *WebhookEvent) eventKey() string {
	// Use event + payment ID as deterministic key
	// YooKassa retries send the same notification with same event+ID
	if w.Object.ID != "" && w.Event != "" {
		return fmt.Sprintf("%s:%s", w.Event, w.Object.ID)
	}
	// Fallback (should not happen with well-formed YooKassa webhook)
	return w.Event
}
