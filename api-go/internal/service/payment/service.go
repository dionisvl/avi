package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/payment/provider"
	paymentrepo "github.com/dionisvl/avi/api-go/internal/repository/payment"
)

type CreatePaymentInput struct {
	UserID    uuid.UUID
	Purpose   model.PaymentPurpose
	SubjectID uuid.UUID
	ReturnURL string
}

type CreatePaymentResult struct {
	ID              uuid.UUID
	Status          model.PaymentStatus
	Amount          model.Money
	ConfirmationURL string
}

type Service interface {
	CreatePayment(ctx context.Context, in CreatePaymentInput) (*CreatePaymentResult, error)
	HandleProviderEvent(ctx context.Context, provider model.PaymentProvider, payload []byte) error
	HasEntitlement(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (bool, error)
}

type service struct {
	repo                      paymentrepo.Repository
	paymentProvider           provider.Provider
	itemReader                ItemReader
	userReader                UserReader
	promoteListingAmountMinor int64
	currency                  string
	logger                    *slog.Logger
}

type ItemReader interface {
	// GetByID returns an item by ID.
	// For promote_listing payments, subject_id is the item.ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.ItemWithDetails, error)
}

type UserReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

func New(
	repo paymentrepo.Repository,
	paymentProvider provider.Provider,
	itemReader ItemReader,
	userReader UserReader,
	promoteListingAmountMinor int64,
	currency string,
	logger *slog.Logger,
) Service {
	return &service{
		repo:                      repo,
		paymentProvider:           paymentProvider,
		itemReader:                itemReader,
		userReader:                userReader,
		promoteListingAmountMinor: promoteListingAmountMinor,
		currency:                  currency,
		logger:                    logger,
	}
}

func (s *service) CreatePayment(ctx context.Context, in CreatePaymentInput) (*CreatePaymentResult, error) {
	// Validate purpose
	if !in.Purpose.Valid() {
		return nil, apperrors.New(apperrors.ErrBadRequest, "invalid payment purpose")
	}

	if in.Purpose != model.PaymentPurposePromoteListing && in.Purpose != model.PaymentPurposeDemoCheckout {
		return nil, apperrors.New(apperrors.ErrBadRequest, "unsupported payment purpose")
	}

	// Validate item exists and user can promote it - for promote_listing, subject_id is the item.ID
	item, err := s.itemReader.GetByID(ctx, in.SubjectID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, apperrors.New(apperrors.ErrNotFound, "item not found")
		}
		return nil, apperrors.New(apperrors.ErrInternal, "failed to load item")
	}

	if in.Purpose == model.PaymentPurposePromoteListing && item.SellerID != in.UserID {
		return nil, apperrors.New(apperrors.ErrForbidden, "you can only promote your own listing")
	}

	hasEntitlement, err := s.repo.HasEntitlement(ctx, in.UserID, in.Purpose, in.SubjectID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "failed to check entitlement")
	}
	if hasEntitlement {
		if in.Purpose == model.PaymentPurposePromoteListing {
			return nil, apperrors.New(apperrors.ErrAlreadyExists, "this listing is already promoted")
		}
		return nil, apperrors.New(apperrors.ErrAlreadyExists, "this demo checkout is already paid")
	}

	// Get buyer email for receipt (54-FZ requirement) BEFORE creating any
	// local payment, so we never leave an orphaned pending row when the buyer
	// has no email.
	user, err := s.userReader.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "failed to load buyer")
	}
	if user == nil || user.Email == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "buyer email is required for a fiscal receipt")
	}
	buyerEmail := user.Email

	// Try to reuse existing pending payment
	existingPending, err := s.repo.GetReusablePending(ctx, in.UserID, in.Purpose, in.SubjectID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "failed to check pending payment")
	}

	if existingPending != nil {
		// If provider payment is complete, reuse it
		if existingPending.ConfirmationURL != "" {
			return &CreatePaymentResult{
				ID:              existingPending.ID,
				Status:          existingPending.Status,
				Amount:          existingPending.Amount,
				ConfirmationURL: existingPending.ConfirmationURL,
			}, nil
		}
		// If payment is stuck in progress, return conflict
		return nil, apperrors.New(apperrors.ErrAlreadyExists, "payment is in progress, please retry")
	}

	// Demo checkout intentionally uses the configured demo amount. The current
	// product model has no order lifecycle yet, and seed listings may use
	// non-RUB prices while YooKassa test payments here are RUB-only.
	amount, err := model.NewMoney(s.promoteListingAmountMinor, s.currency)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "invalid payment amount configuration")
	}
	payment := model.NewPayment(in.UserID, in.Purpose, in.SubjectID, amount)

	// Try to insert locally
	err = s.repo.Create(ctx, payment)
	if err != nil {
		if errors.Is(err, paymentrepo.ErrConflictPending) {
			// Lost race, re-read and apply reuse logic
			existingPending, reReadErr := s.repo.GetReusablePending(ctx, in.UserID, in.Purpose, in.SubjectID)
			if reReadErr != nil {
				return nil, apperrors.New(apperrors.ErrInternal, "failed to re-read pending payment")
			}
			if existingPending != nil && existingPending.ConfirmationURL != "" {
				return &CreatePaymentResult{
					ID:              existingPending.ID,
					Status:          existingPending.Status,
					Amount:          existingPending.Amount,
					ConfirmationURL: existingPending.ConfirmationURL,
				}, nil
			}
			return nil, apperrors.New(apperrors.ErrAlreadyExists, "payment is in progress, please retry")
		}
		return nil, apperrors.New(apperrors.ErrInternal, "failed to create payment")
	}

	// Call provider through port
	providerInput := provider.CreatePaymentInput{
		LocalPaymentID: payment.ID,
		UserID:         in.UserID,
		Purpose:        in.Purpose,
		SubjectID:      in.SubjectID,
		Amount:         amount,
		Description:    descriptionFromPurpose(in.Purpose),
		ReturnURL:      in.ReturnURL,
		IdempotencyKey: payment.IdempotencyKey,
		BuyerEmail:     buyerEmail,
	}

	created, err := s.paymentProvider.CreatePayment(ctx, providerInput)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "failed to create provider payment")
	}

	// Persist provider payment ID and confirmation URL
	err = s.repo.SetProviderCreated(ctx, payment.ID, created.ProviderPaymentID, created.ConfirmationURL, created.Metadata, created.Receipt)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "failed to persist provider payment")
	}

	return &CreatePaymentResult{
		ID:              payment.ID,
		Status:          created.Status,
		Amount:          amount,
		ConfirmationURL: created.ConfirmationURL,
	}, nil
}

func descriptionFromPurpose(purpose model.PaymentPurpose) string {
	switch purpose {
	case model.PaymentPurposeDemoCheckout:
		return "Avi demo checkout"
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

func (s *service) HandleProviderEvent(ctx context.Context, providerType model.PaymentProvider, payload []byte) error {
	// Parse webhook event based on provider type
	if providerType != model.PaymentProviderYooKassa {
		return fmt.Errorf("unsupported provider: %s", providerType)
	}

	event, err := s.paymentProvider.ParseWebhookEvent(payload)
	if err != nil {
		// Malformed payload — log and return 200 so provider stops retrying
		return nil
	}

	verifiedPayment, err := s.paymentProvider.GetPayment(ctx, event.ProviderPaymentID)
	if err != nil {
		return fmt.Errorf("failed to verify provider payment status: %w", err)
	}
	if verifiedPayment == nil {
		return fmt.Errorf("provider payment status is empty")
	}
	if verifiedPayment.ProviderPaymentID != event.ProviderPaymentID {
		return fmt.Errorf("provider payment id mismatch: webhook=%s provider=%s", event.ProviderPaymentID, verifiedPayment.ProviderPaymentID)
	}
	if verifiedPayment.ProviderStatus != event.ProviderStatus {
		// A forged or stale webhook must not poison the idempotency table for a future real event.
		return nil
	}

	// Create event record for idempotency
	paymentEvent := &paymentrepo.PaymentEvent{
		ID:                uuid.New(),
		Provider:          providerType,
		EventType:         event.EventType,
		ProviderPaymentID: event.ProviderPaymentID,
		EventKey:          event.EventKey,
		Status:            "pending",
		Payload:           payload,
		CreatedAt:         time.Now(),
	}

	created, err := s.repo.CreateEvent(ctx, paymentEvent)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	// If this is a duplicate webhook, return success
	if !created {
		return nil
	}

	// Load local payment by provider payment ID
	payment, err := s.repo.GetByProviderPaymentID(ctx, event.ProviderPaymentID)
	if err != nil {
		// No local payment for this provider id. A retry cannot fix this, so
		// mark the event ignored and return 200 — otherwise the provider would
		// retry forever. The event row stays as an audit trail.
		s.logger.Warn("webhook for unknown local payment",
			slog.String("provider_payment_id", event.ProviderPaymentID),
			slog.String("error", err.Error()),
		)
		_ = s.repo.MarkEventIgnored(ctx, paymentEvent.ID)
		return nil
	}

	// Process based on provider-confirmed status, not the public webhook payload.
	switch verifiedPayment.Status {
	case model.PaymentStatusSucceeded:
		// Create entitlement and mark payment succeeded atomically
		now := time.Now()
		entitlement := &paymentrepo.PaidEntitlement{
			ID:        uuid.New(),
			UserID:    payment.UserID,
			Purpose:   payment.Purpose,
			SubjectID: payment.SubjectID,
			PaymentID: payment.ID,
			StartsAt:  now,
			CreatedAt: now,
		}

		err := s.repo.ApplySucceeded(ctx, payment.ID, paymentEvent.ID, verifiedPayment.Metadata, now, entitlement)
		if err != nil {
			_ = s.repo.MarkEventFailed(ctx, paymentEvent.ID, err.Error())
			return fmt.Errorf("failed to apply succeeded: %w", err)
		}

	case model.PaymentStatusCanceled:
		// Mark payment canceled atomically with event processing
		now := time.Now()
		err := s.repo.ApplyCanceled(ctx, payment.ID, paymentEvent.ID, verifiedPayment.Metadata, now)
		if err != nil {
			_ = s.repo.MarkEventFailed(ctx, paymentEvent.ID, err.Error())
			return fmt.Errorf("failed to apply canceled: %w", err)
		}

	case model.PaymentStatusPending, model.PaymentStatusRefunded, model.PaymentStatusPartiallyRefunded:
		_ = s.repo.MarkEventIgnored(ctx, paymentEvent.ID)

	default:
		// Unknown status — mark event as ignored
		_ = s.repo.MarkEventIgnored(ctx, paymentEvent.ID)
	}

	return nil
}

func (s *service) HasEntitlement(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (bool, error) {
	return s.repo.HasEntitlement(ctx, userID, purpose, subjectID)
}
