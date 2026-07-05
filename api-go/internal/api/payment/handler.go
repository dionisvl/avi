package payment

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dionisvl/avi/api-go/internal/api"
	apimiddleware "github.com/dionisvl/avi/api-go/internal/api/middleware"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/model"
	paymentservice "github.com/dionisvl/avi/api-go/internal/service/payment"
)

const maxWebhookBodyBytes = 64 << 10 // 64 KiB — YooKassa notifications are small

type Handler struct {
	svc            paymentservice.Service
	returnURL      string
	frontendDomain string
	logger         *slog.Logger
}

func NewHandler(svc paymentservice.Service, returnURL, frontendDomain string, logger *slog.Logger) *Handler {
	return &Handler{
		svc:            svc,
		returnURL:      returnURL,
		frontendDomain: frontendDomain,
		logger:         logger,
	}
}

func (h *Handler) Routes(authSvc apimiddleware.TokenValidator) chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(apimiddleware.AuthRequired(authSvc))
		r.Post("/", h.createPayment)
	})

	r.Post("/webhooks/yookassa", h.webhookYookassa)

	return r
}

// Request/Response types

type CreatePaymentRequest struct {
	Purpose   string `json:"purpose" validate:"required,oneof=promote_listing demo_checkout"`
	SubjectID string `json:"subject_id" validate:"required,uuid"`
	ReturnURL string `json:"return_url"`
}

type AmountResponse struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type CreatePaymentResponse struct {
	ID              uuid.UUID      `json:"id"`
	Status          string         `json:"status"`
	Amount          AmountResponse `json:"amount"`
	ConfirmationURL string         `json:"confirmation_url"`
}

type WebhookResponse struct {
	Processed bool `json:"processed"`
}

// @Summary Create a payment
// @Tags payments
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body CreatePaymentRequest true "Request"
// @Success 200 {object} CreatePaymentResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 409 {object} api.ProblemDetails
// @Router /api/v1/payments [post]
func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(apimiddleware.ContextKeyUserID).(uuid.UUID)
	if !ok {
		api.Error(w, h.logger, apierr.New(apierr.ErrInternal, "user not authenticated"))
		return
	}

	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "invalid request body"))
		return
	}

	if err := api.ValidateStruct(&req); err != nil {
		validErr := apierr.New(apierr.ErrValidation, err.Error())
		api.Error(w, h.logger, validErr)
		return
	}

	subjectID, err := uuid.Parse(req.SubjectID)
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "invalid subject_id"))
		return
	}

	purpose := model.PaymentPurpose(req.Purpose)
	returnURL := h.returnURL
	if req.ReturnURL != "" {
		if !h.isAllowedReturnURL(req.ReturnURL) {
			api.Error(w, h.logger, apierr.New(apierr.ErrBadRequest, "return_url host is not allowed"))
			return
		}
		returnURL = req.ReturnURL
	}

	svcInput := paymentservice.CreatePaymentInput{
		UserID:    userID,
		Purpose:   purpose,
		SubjectID: subjectID,
		ReturnURL: returnURL,
	}

	result, err := h.svc.CreatePayment(ctx, svcInput)
	if err != nil {
		if appErr, ok := err.(*apierr.AppError); ok {
			api.Error(w, h.logger, appErr)
		} else {
			api.Error(w, h.logger, apierr.New(apierr.ErrInternal, "failed to create payment"))
		}
		return
	}

	response := CreatePaymentResponse{
		ID:              result.ID,
		Status:          string(result.Status),
		ConfirmationURL: result.ConfirmationURL,
		Amount: AmountResponse{
			Value:    result.Amount.Format(),
			Currency: result.Amount.Currency(),
		},
	}

	api.JSON(w, http.StatusOK, response)
}

// isAllowedReturnURL accepts only http(s) URLs whose host matches the host of
// the configured default return URL or the trusted frontend domain. This
// prevents clients from setting an arbitrary post-payment redirect target.
func (h *Handler) isAllowedReturnURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	for _, allowed := range []string{h.returnURL, h.frontendDomain} {
		if allowed == "" {
			continue
		}
		if au, err := url.Parse(allowed); err == nil && au.Host != "" && strings.EqualFold(au.Host, u.Host) {
			return true
		}
	}
	return false
}

// @Summary Webhook for YooKassa payment notifications
// @Tags payments
// @Accept json
// @Produce json
// @Success 200 {object} WebhookResponse
// @Failure 500 {object} api.ProblemDetails
// @Router /api/v1/payments/webhooks/yookassa [post]
func (h *Handler) webhookYookassa(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", "error", err)
		api.JSON(w, http.StatusOK, WebhookResponse{Processed: false})
		return
	}

	err = h.svc.HandleProviderEvent(ctx, model.PaymentProviderYooKassa, body)
	if err != nil {
		h.logger.Error("failed to handle provider event", "error", err)
		api.Error(w, h.logger, apierr.New(apierr.ErrInternal, "failed to handle provider event"))
		return
	}

	api.JSON(w, http.StatusOK, WebhookResponse{Processed: true})
}
