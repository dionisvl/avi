package payment_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/payment/provider"
	paymentrepo "github.com/dionisvl/avi/api-go/internal/repository/payment"
	"github.com/dionisvl/avi/api-go/internal/service/payment"
)

type fakeItemReader struct {
	sellerID uuid.UUID
}

func (f *fakeItemReader) GetByID(ctx context.Context, id uuid.UUID) (*model.ItemWithDetails, error) {
	item := &model.ItemWithDetails{}
	item.ID = id
	item.SellerID = f.sellerID
	return item, nil
}

type fakeUserReader struct {
	email string
}

func (f *fakeUserReader) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	email := f.email
	if email == "" {
		email = "test@example.com"
	}
	return &model.User{Email: email}, nil
}

// fakeUserReaderNoEmail returns a user without an email, to exercise the
// 54-FZ receipt requirement.
type fakeUserReaderNoEmail struct{}

func (f *fakeUserReaderNoEmail) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return &model.User{Email: ""}, nil
}

type fakeProvider struct {
	created          *provider.CreatedPayment
	paymentInfo      *provider.PaymentInfo
	err              error
	getPaymentErr    error
	lastIn           provider.CreatePaymentInput
	lastGetPaymentID string
}

func (f *fakeProvider) CreatePayment(ctx context.Context, in provider.CreatePaymentInput) (*provider.CreatedPayment, error) {
	f.lastIn = in
	if f.err != nil {
		return nil, f.err
	}
	return f.created, nil
}

func (f *fakeProvider) GetPayment(ctx context.Context, providerPaymentID string) (*provider.PaymentInfo, error) {
	f.lastGetPaymentID = providerPaymentID
	if f.getPaymentErr != nil {
		return nil, f.getPaymentErr
	}
	return f.paymentInfo, nil
}

func (f *fakeProvider) ParseWebhookEvent(payload []byte) (*provider.WebhookEvent, error) {
	var event struct {
		Event  string `json:"event"`
		Object struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"object"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	return &provider.WebhookEvent{
		ProviderPaymentID: event.Object.ID,
		EventType:         event.Event,
		ProviderStatus:    event.Object.Status,
		EventKey:          event.Event + ":" + event.Object.ID,
	}, nil
}

type fakeRepository struct {
	payments                 map[uuid.UUID]*model.Payment
	pendingBySubject         map[string]*model.Payment
	entitlements             map[string]bool
	events                   map[uuid.UUID]*paymentrepo.PaymentEvent
	lastCreatedPayment       *model.Payment
	lastSetProviderCreated   *model.Payment
	lastApplySucceeded       *model.Payment
	lastApplyCanceledPayment *model.Payment
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		payments:         make(map[uuid.UUID]*model.Payment),
		pendingBySubject: make(map[string]*model.Payment),
		entitlements:     make(map[string]bool),
		events:           make(map[uuid.UUID]*paymentrepo.PaymentEvent),
	}
}

func (f *fakeRepository) Create(ctx context.Context, p *model.Payment) error {
	f.lastCreatedPayment = p
	f.payments[p.ID] = p
	key := p.UserID.String() + ":" + p.Purpose.String() + ":" + p.SubjectID.String()
	f.pendingBySubject[key] = p
	return nil
}

func (f *fakeRepository) SetProviderCreated(ctx context.Context, id uuid.UUID, providerPaymentID, confirmationURL string, providerMetadata map[string]any, receipt json.RawMessage) error {
	p := f.payments[id]
	if p == nil {
		return paymentrepo.ErrPaymentNotFound
	}
	p.ProviderPaymentID = providerPaymentID
	p.ConfirmationURL = confirmationURL
	f.lastSetProviderCreated = p
	return nil
}

func (f *fakeRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	p := f.payments[id]
	if p == nil {
		return nil, paymentrepo.ErrPaymentNotFound
	}
	return p, nil
}

func (f *fakeRepository) GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*model.Payment, error) {
	for _, p := range f.payments {
		if p.ProviderPaymentID == providerPaymentID {
			return p, nil
		}
	}
	return nil, paymentrepo.ErrPaymentNotFound
}

func (f *fakeRepository) GetReusablePending(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (*model.Payment, error) {
	key := userID.String() + ":" + purpose.String() + ":" + subjectID.String()
	return f.pendingBySubject[key], nil
}

func (f *fakeRepository) CreateEvent(ctx context.Context, event *paymentrepo.PaymentEvent) (bool, error) {
	for _, existing := range f.events {
		if existing.Provider == event.Provider && existing.EventKey == event.EventKey {
			return false, nil
		}
	}
	f.events[event.ID] = event
	return true, nil
}

func (f *fakeRepository) MarkEventIgnored(ctx context.Context, id uuid.UUID) error {
	e := f.events[id]
	if e == nil {
		return paymentrepo.ErrEventNotFound
	}
	e.Status = "ignored"
	return nil
}

func (f *fakeRepository) MarkEventFailed(ctx context.Context, id uuid.UUID, message string) error {
	e := f.events[id]
	if e == nil {
		return paymentrepo.ErrEventNotFound
	}
	e.Status = "failed"
	e.ErrorMessage = &message
	return nil
}

func (f *fakeRepository) ApplySucceeded(ctx context.Context, paymentID, eventID uuid.UUID, providerMetadata map[string]any, paidAt time.Time, entitlement *paymentrepo.PaidEntitlement) error {
	p := f.payments[paymentID]
	if p == nil {
		return paymentrepo.ErrPaymentNotFound
	}
	p.Status = model.PaymentStatusSucceeded
	p.PaidAt = &paidAt
	f.lastApplySucceeded = p

	key := entitlement.UserID.String() + ":" + entitlement.Purpose.String() + ":" + entitlement.SubjectID.String()
	f.entitlements[key] = true

	e := f.events[eventID]
	if e != nil {
		e.Status = "processed"
	}
	return nil
}

func (f *fakeRepository) ApplyCanceled(ctx context.Context, paymentID, eventID uuid.UUID, providerMetadata map[string]any, canceledAt time.Time) error {
	p := f.payments[paymentID]
	if p == nil {
		return paymentrepo.ErrPaymentNotFound
	}
	p.Status = model.PaymentStatusCanceled
	p.CanceledAt = &canceledAt
	f.lastApplyCanceledPayment = p

	e := f.events[eventID]
	if e != nil {
		e.Status = "processed"
	}
	return nil
}

func (f *fakeRepository) HasEntitlement(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (bool, error) {
	key := userID.String() + ":" + purpose.String() + ":" + subjectID.String()
	return f.entitlements[key], nil
}

func (f *fakeRepository) HasEntitlementSet(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := make(map[uuid.UUID]bool, len(subjectIDs))
	for _, id := range subjectIDs {
		key := userID.String() + ":" + purpose.String() + ":" + id.String()
		if f.entitlements[key] {
			result[id] = true
		}
	}
	return result, nil
}

func TestService_CreatePayment_PromoteListingCreatesProviderPayment(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	itemID := uuid.New()

	itemReader := &fakeItemReader{sellerID: userID}
	fakeRepo := newFakeRepository()

	expectedCreated := &provider.CreatedPayment{
		ProviderPaymentID: "yoo-" + uuid.New().String(),
		Status:            model.PaymentStatusPending,
		ConfirmationURL:   "https://yookassa.ru/confirm",
		Metadata:          map[string]any{},
	}
	fakeProvider := &fakeProvider{created: expectedCreated}

	svc := payment.New(fakeRepo, fakeProvider, itemReader, &fakeUserReader{}, 10000, "RUB", slog.Default())

	result, err := svc.CreatePayment(ctx, payment.CreatePaymentInput{
		UserID:    userID,
		Purpose:   model.PaymentPurposePromoteListing,
		SubjectID: itemID,
		ReturnURL: "https://example.com/thank-you",
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, model.PaymentStatusPending, result.Status)
	assert.Equal(t, expectedCreated.ConfirmationURL, result.ConfirmationURL)
	assert.Equal(t, "100.00", result.Amount.Format())

	// Verify provider was called
	assert.Equal(t, model.PaymentPurposePromoteListing, fakeProvider.lastIn.Purpose)
	assert.Equal(t, int64(10000), fakeProvider.lastIn.Amount.AmountMinor())

	// Verify payment was stored
	assert.NotNil(t, fakeRepo.lastCreatedPayment)
	assert.Equal(t, userID, fakeRepo.lastCreatedPayment.UserID)

	// Verify provider payment ID was persisted
	assert.NotNil(t, fakeRepo.lastSetProviderCreated)
	assert.Equal(t, expectedCreated.ProviderPaymentID, fakeRepo.lastSetProviderCreated.ProviderPaymentID)
}

func TestService_CreatePayment_DemoCheckoutAllowsNonOwner(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	sellerID := uuid.New()
	itemID := uuid.New()

	fakeRepo := newFakeRepository()
	expectedCreated := &provider.CreatedPayment{
		ProviderPaymentID: "yoo-" + uuid.New().String(),
		Status:            model.PaymentStatusPending,
		ConfirmationURL:   "https://yookassa.ru/confirm",
		Metadata:          map[string]any{},
	}
	fakeProvider := &fakeProvider{created: expectedCreated}

	svc := payment.New(
		fakeRepo,
		fakeProvider,
		&fakeItemReader{sellerID: sellerID},
		&fakeUserReader{},
		10000,
		"RUB",
		slog.Default(),
	)

	result, err := svc.CreatePayment(ctx, payment.CreatePaymentInput{
		UserID:    userID,
		Purpose:   model.PaymentPurposeDemoCheckout,
		SubjectID: itemID,
		ReturnURL: "https://example.com/thank-you",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expectedCreated.ConfirmationURL, result.ConfirmationURL)
	assert.Equal(t, model.PaymentPurposeDemoCheckout, fakeRepo.lastCreatedPayment.Purpose)
	assert.Equal(t, model.PaymentPurposeDemoCheckout, fakeProvider.lastIn.Purpose)
	assert.Equal(t, "Avi demo checkout", fakeProvider.lastIn.Description)
}

func TestService_CreatePayment_NoBuyerEmailIsRejected(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	itemID := uuid.New()

	itemReader := &fakeItemReader{sellerID: userID}
	fakeRepo := newFakeRepository()
	fakeProvider := &fakeProvider{created: &provider.CreatedPayment{}}

	svc := payment.New(fakeRepo, fakeProvider, itemReader, &fakeUserReaderNoEmail{}, 10000, "RUB", slog.Default())

	result, err := svc.CreatePayment(ctx, payment.CreatePaymentInput{
		UserID:    userID,
		Purpose:   model.PaymentPurposePromoteListing,
		SubjectID: itemID,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	// No local payment created and provider never called — no orphaned pending row.
	assert.Nil(t, fakeRepo.lastCreatedPayment)
	assert.Empty(t, fakeProvider.lastIn.IdempotencyKey)
}

func TestService_CreatePayment_PromoteListingAlreadyPromotedReturnsConflict(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	itemID := uuid.New()

	itemReader := &fakeItemReader{sellerID: userID}
	fakeRepo := newFakeRepository()

	// Pre-populate entitlement
	entKey := userID.String() + ":" + model.PaymentPurposePromoteListing.String() + ":" + itemID.String()
	fakeRepo.entitlements[entKey] = true

	fakeProvider := &fakeProvider{}
	svc := payment.New(fakeRepo, fakeProvider, itemReader, &fakeUserReader{}, 10000, "RUB", slog.Default())

	result, err := svc.CreatePayment(ctx, payment.CreatePaymentInput{
		UserID:    userID,
		Purpose:   model.PaymentPurposePromoteListing,
		SubjectID: itemID,
		ReturnURL: "https://example.com/thank-you",
	})

	// Should return conflict error
	assert.Error(t, err)
	assert.Nil(t, result)

	// Provider should not be called
	assert.Equal(t, (provider.CreatePaymentInput{}), fakeProvider.lastIn)
}

func TestService_HasEntitlement(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	itemID := uuid.New()

	itemReader := &fakeItemReader{sellerID: userID}
	fakeRepo := newFakeRepository()

	// Pre-populate entitlement
	entKey := userID.String() + ":" + model.PaymentPurposePromoteListing.String() + ":" + itemID.String()
	fakeRepo.entitlements[entKey] = true

	fakeProvider := &fakeProvider{}
	svc := payment.New(fakeRepo, fakeProvider, itemReader, &fakeUserReader{}, 10000, "RUB", slog.Default())

	hasEnt, err := svc.HasEntitlement(ctx, userID, model.PaymentPurposePromoteListing, itemID)
	require.NoError(t, err)
	assert.True(t, hasEnt)

	// Different user should not have entitlement
	otherUserID := uuid.New()
	hasEnt2, err := svc.HasEntitlement(ctx, otherUserID, model.PaymentPurposePromoteListing, itemID)
	require.NoError(t, err)
	assert.False(t, hasEnt2)
}

func TestService_HandleProviderEvent_VerifiesProviderStatusBeforeGrantingEntitlement(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	itemID := uuid.New()
	providerPaymentID := "yoo-" + uuid.New().String()

	fakeRepo := newFakeRepository()
	localPayment := model.NewPayment(userID, model.PaymentPurposePromoteListing, itemID, mustMoney(t, 10000))
	localPayment.ProviderPaymentID = providerPaymentID
	fakeRepo.payments[localPayment.ID] = localPayment

	fakeProvider := &fakeProvider{
		paymentInfo: &provider.PaymentInfo{
			ProviderPaymentID: providerPaymentID,
			Status:            model.PaymentStatusSucceeded,
			ProviderStatus:    "succeeded",
			Metadata:          map[string]any{"provider_status": "succeeded"},
		},
	}
	svc := payment.New(fakeRepo, fakeProvider, &fakeItemReader{sellerID: userID}, &fakeUserReader{}, 10000, "RUB", slog.Default())

	err := svc.HandleProviderEvent(ctx, model.PaymentProviderYooKassa, yookassaWebhookPayload(t, "payment.succeeded", providerPaymentID, "succeeded"))

	require.NoError(t, err)
	assert.Equal(t, providerPaymentID, fakeProvider.lastGetPaymentID)
	assert.Equal(t, model.PaymentStatusSucceeded, localPayment.Status)
	assert.NotNil(t, fakeRepo.lastApplySucceeded)

	entitlementKey := userID.String() + ":" + model.PaymentPurposePromoteListing.String() + ":" + itemID.String()
	assert.True(t, fakeRepo.entitlements[entitlementKey])
}

func TestService_HandleProviderEvent_DoesNotTrustSpoofedSucceededPayload(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	itemID := uuid.New()
	providerPaymentID := "yoo-" + uuid.New().String()

	fakeRepo := newFakeRepository()
	localPayment := model.NewPayment(userID, model.PaymentPurposePromoteListing, itemID, mustMoney(t, 10000))
	localPayment.ProviderPaymentID = providerPaymentID
	fakeRepo.payments[localPayment.ID] = localPayment

	fakeProvider := &fakeProvider{
		paymentInfo: &provider.PaymentInfo{
			ProviderPaymentID: providerPaymentID,
			Status:            model.PaymentStatusPending,
			ProviderStatus:    "pending",
			Metadata:          map[string]any{"provider_status": "pending"},
		},
	}
	svc := payment.New(fakeRepo, fakeProvider, &fakeItemReader{sellerID: userID}, &fakeUserReader{}, 10000, "RUB", slog.Default())

	err := svc.HandleProviderEvent(ctx, model.PaymentProviderYooKassa, yookassaWebhookPayload(t, "payment.succeeded", providerPaymentID, "succeeded"))

	require.NoError(t, err)
	assert.Equal(t, providerPaymentID, fakeProvider.lastGetPaymentID)
	assert.Equal(t, model.PaymentStatusPending, localPayment.Status)
	assert.Nil(t, fakeRepo.lastApplySucceeded)
	assert.Empty(t, fakeRepo.events)
}

func mustMoney(t *testing.T, amountMinor int64) model.Money {
	t.Helper()

	money, err := model.NewMoney(amountMinor, "RUB")
	require.NoError(t, err)
	return money
}

func yookassaWebhookPayload(t *testing.T, event, providerPaymentID, status string) []byte {
	t.Helper()

	payload := map[string]any{
		"type":  "notification",
		"event": event,
		"object": map[string]any{
			"id":     providerPaymentID,
			"status": status,
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)
	return data
}
