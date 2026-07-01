package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dionisvl/avi/api-go/internal/model"
	"github.com/dionisvl/avi/api-go/internal/platform/dbtx"
)

var (
	ErrPaymentNotFound = fmt.Errorf("payment not found")
	ErrEventNotFound   = fmt.Errorf("payment event not found")
	ErrConflictPending = fmt.Errorf("payment already pending for this subject")
)

type PaidEntitlement struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Purpose   model.PaymentPurpose
	SubjectID uuid.UUID
	PaymentID uuid.UUID
	StartsAt  time.Time
	ExpiresAt *time.Time
	CreatedAt time.Time
}

type PaymentEvent struct {
	ID                uuid.UUID
	Provider          model.PaymentProvider
	EventType         string
	ProviderPaymentID string
	EventKey          string
	Status            string // pending, processed, ignored, failed
	Payload           []byte
	ErrorMessage      *string
	ProcessedAt       *time.Time
	CreatedAt         time.Time
}

type Repository interface {
	Create(ctx context.Context, payment *model.Payment) error
	SetProviderCreated(ctx context.Context, id uuid.UUID, providerPaymentID, confirmationURL string, providerMetadata map[string]any, receipt json.RawMessage) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Payment, error)
	GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*model.Payment, error)
	GetReusablePending(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (*model.Payment, error)

	CreateEvent(ctx context.Context, event *PaymentEvent) (created bool, err error)
	MarkEventIgnored(ctx context.Context, id uuid.UUID) error
	MarkEventFailed(ctx context.Context, id uuid.UUID, message string) error

	ApplySucceeded(ctx context.Context, paymentID, eventID uuid.UUID, providerMetadata map[string]any, paidAt time.Time, entitlement *PaidEntitlement) error
	ApplyCanceled(ctx context.Context, paymentID, eventID uuid.UUID, providerMetadata map[string]any, canceledAt time.Time) error

	HasEntitlement(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (bool, error)
	HasEntitlementSet(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type repository struct {
	db dbtx.DB
}

func New(db dbtx.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, payment *model.Payment) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO payments (id, user_id, purpose, subject_id, amount_minor, currency, status, provider, idempotency_key, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		payment.ID, payment.UserID, string(payment.Purpose), payment.SubjectID,
		payment.Amount.AmountMinor(), payment.Amount.Currency(),
		string(payment.Status), string(payment.Provider),
		payment.IdempotencyKey, payment.CreatedAt, payment.UpdatedAt,
	).Scan(&payment.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "payments_one_pending_per_subject_uidx" {
			return ErrConflictPending
		}
		return err
	}
	return nil
}

func (r *repository) SetProviderCreated(ctx context.Context, id uuid.UUID, providerPaymentID, confirmationURL string, providerMetadata map[string]any, receipt json.RawMessage) error {
	metaJSON, err := json.Marshal(providerMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var receiptArg any
	if len(receipt) > 0 {
		receiptArg = []byte(receipt)
	}

	result, err := r.db.Exec(ctx,
		`UPDATE payments SET provider_payment_id = $1, confirmation_url = $2, provider_metadata = $3, receipt = $4, updated_at = NOW()
		 WHERE id = $5`,
		providerPaymentID, confirmationURL, metaJSON, receiptArg, id,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	return r.scanPayment(ctx,
		`SELECT id, user_id, purpose, subject_id, amount_minor, currency, status, provider, provider_payment_id, confirmation_url, idempotency_key, provider_method_id, paid_at, canceled_at, created_at, updated_at
		 FROM payments WHERE id = $1`,
		id,
	)
}

func (r *repository) GetByProviderPaymentID(ctx context.Context, providerPaymentID string) (*model.Payment, error) {
	return r.scanPayment(ctx,
		`SELECT id, user_id, purpose, subject_id, amount_minor, currency, status, provider, provider_payment_id, confirmation_url, idempotency_key, provider_method_id, paid_at, canceled_at, created_at, updated_at
		 FROM payments WHERE provider_payment_id = $1`,
		providerPaymentID,
	)
}

func (r *repository) GetReusablePending(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (*model.Payment, error) {
	payment, err := r.scanPayment(ctx,
		`SELECT id, user_id, purpose, subject_id, amount_minor, currency, status, provider, provider_payment_id, confirmation_url, idempotency_key, provider_method_id, paid_at, canceled_at, created_at, updated_at
		 FROM payments
		 WHERE user_id = $1 AND purpose = $2 AND subject_id = $3 AND status = 'pending'
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID, string(purpose), subjectID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return payment, err
}

func (r *repository) scanPayment(ctx context.Context, query string, args ...any) (*model.Payment, error) {
	var payment model.Payment
	var purpose, status string
	var amountMinor int64
	var currency string
	var provider string
	// provider_payment_id and confirmation_url are nullable until SetProviderCreated runs.
	// Use *string to handle NULL without scan errors on the reuse/conflict path.
	var providerPaymentID *string
	var confirmationURL *string

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&payment.ID, &payment.UserID, &purpose, &payment.SubjectID,
		&amountMinor, &currency, &status, &provider,
		&providerPaymentID, &confirmationURL, &payment.IdempotencyKey,
		&payment.ProviderMethodID, &payment.PaidAt, &payment.CanceledAt,
		&payment.CreatedAt, &payment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert nullable strings to empty strings for the model
	if providerPaymentID != nil {
		payment.ProviderPaymentID = *providerPaymentID
	}
	if confirmationURL != nil {
		payment.ConfirmationURL = *confirmationURL
	}

	payment.Purpose = model.PaymentPurpose(purpose)
	payment.Status = model.PaymentStatus(status)
	payment.Provider = model.PaymentProvider(provider)
	payment.Amount, err = model.NewMoney(amountMinor, currency)
	if err != nil {
		return nil, fmt.Errorf("invalid money: %w", err)
	}

	return &payment, nil
}

func (r *repository) CreateEvent(ctx context.Context, event *PaymentEvent) (created bool, err error) {
	var insertedID uuid.UUID
	err = r.db.QueryRow(ctx,
		`INSERT INTO payment_events (id, provider, event_type, provider_payment_id, event_key, status, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (provider, event_key) DO NOTHING
		 RETURNING id`,
		event.ID, string(event.Provider), event.EventType, event.ProviderPaymentID,
		event.EventKey, event.Status, event.Payload, event.CreatedAt,
	).Scan(&insertedID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Conflict on (provider, event_key) — duplicate webhook, nothing inserted.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *repository) MarkEventIgnored(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx,
		`UPDATE payment_events SET status = 'ignored' WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrEventNotFound
	}
	return nil
}

func (r *repository) MarkEventFailed(ctx context.Context, id uuid.UUID, message string) error {
	result, err := r.db.Exec(ctx,
		`UPDATE payment_events SET status = 'failed', error_message = $1 WHERE id = $2`,
		message, id,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrEventNotFound
	}
	return nil
}

func (r *repository) ApplySucceeded(ctx context.Context, paymentID, eventID uuid.UUID, providerMetadata map[string]any, paidAt time.Time, entitlement *PaidEntitlement) error {
	tb, ok := r.db.(dbtx.TxBeginner)
	if !ok {
		return fmt.Errorf("database does not support transactions")
	}

	tx, err := tb.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Update payment status
	metaJSON, _ := json.Marshal(providerMetadata)
	result, err := tx.Exec(ctx,
		`UPDATE payments SET status = 'succeeded', paid_at = $1, provider_metadata = $2, updated_at = NOW() WHERE id = $3`,
		paidAt, metaJSON, paymentID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrPaymentNotFound
	}

	// Create entitlement
	_, err = tx.Exec(ctx,
		`INSERT INTO paid_entitlements (id, user_id, purpose, subject_id, payment_id, starts_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, purpose, subject_id) WHERE expires_at IS NULL DO NOTHING`,
		entitlement.ID, entitlement.UserID, string(entitlement.Purpose), entitlement.SubjectID,
		entitlement.PaymentID, entitlement.StartsAt, entitlement.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Mark event processed
	result, err = tx.Exec(ctx,
		`UPDATE payment_events SET status = 'processed', processed_at = NOW() WHERE id = $1`,
		eventID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrEventNotFound
	}

	return tx.Commit(ctx)
}

func (r *repository) ApplyCanceled(ctx context.Context, paymentID, eventID uuid.UUID, providerMetadata map[string]any, canceledAt time.Time) error {
	tb, ok := r.db.(dbtx.TxBeginner)
	if !ok {
		return fmt.Errorf("database does not support transactions")
	}

	tx, err := tb.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Check current status before updating
	var currentStatus string
	err = tx.QueryRow(ctx,
		`SELECT status FROM payments WHERE id = $1`,
		paymentID,
	).Scan(&currentStatus)
	if err != nil {
		return err
	}

	// Do not flip succeeded back to canceled
	if currentStatus == string(model.PaymentStatusSucceeded) {
		// Just mark event as ignored and return
		_, _ = tx.Exec(ctx,
			`UPDATE payment_events SET status = 'ignored' WHERE id = $1`,
			eventID,
		)
		return tx.Commit(ctx)
	}

	// Update payment status
	metaJSON, _ := json.Marshal(providerMetadata)
	result, err := tx.Exec(ctx,
		`UPDATE payments SET status = 'canceled', canceled_at = $1, provider_metadata = $2, updated_at = NOW()
		 WHERE id = $3 AND status != 'succeeded'`,
		canceledAt, metaJSON, paymentID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		// Payment is already succeeded or not found
		_, _ = tx.Exec(ctx,
			`UPDATE payment_events SET status = 'ignored' WHERE id = $1`,
			eventID,
		)
		return tx.Commit(ctx)
	}

	// Mark event processed
	_, err = tx.Exec(ctx,
		`UPDATE payment_events SET status = 'processed', processed_at = NOW() WHERE id = $1`,
		eventID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *repository) HasEntitlement(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM paid_entitlements
		 WHERE user_id = $1 AND purpose = $2 AND subject_id = $3 AND expires_at IS NULL)`,
		userID, string(purpose), subjectID,
	).Scan(&exists)
	return exists, err
}

func (r *repository) HasEntitlementSet(ctx context.Context, userID uuid.UUID, purpose model.PaymentPurpose, subjectIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := make(map[uuid.UUID]bool, len(subjectIDs))
	if len(subjectIDs) == 0 {
		return result, nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT subject_id FROM paid_entitlements
		 WHERE user_id = $1 AND purpose = $2 AND subject_id = ANY($3) AND expires_at IS NULL`,
		userID, string(purpose), subjectIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}
