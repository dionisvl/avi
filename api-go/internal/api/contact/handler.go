package contact

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dionisvl/avi/api-go/internal/api"
	"github.com/dionisvl/avi/api-go/internal/platform/locale"
	contactservice "github.com/dionisvl/avi/api-go/internal/service/contact"
)

type Handler struct {
	svc    contactservice.Service
	logger *slog.Logger
}

func NewHandler(svc contactservice.Service, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.create)
	return r
}

type CreateRequest struct {
	Name    string `json:"name"    validate:"required,min=2,max=120"`
	Email   string `json:"email"   validate:"required,email,max=254"`
	Subject string `json:"subject" validate:"omitempty,max=160"`
	Message string `json:"message" validate:"required,min=10,max=4000"`
}

type CreateResponse struct {
	Status string `json:"status"`
}

type SingleResponse struct {
	Data CreateResponse `json:"data"`
}

// @Summary Send contact form message
// @Tags contact-messages
// @Accept json
// @Produce json
// @Param body body CreateRequest true "Contact message"
// @Success 202 {object} SingleResponse
// @Failure 400 {object} api.ProblemDetails
// @Failure 429 {object} api.ProblemDetails
// @Failure 500 {object} api.ProblemDetails
// @Router /api/v1/contact-messages [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := api.DecodeAndValidate[CreateRequest](r)
	if err != nil {
		api.Error(w, h.logger, api.AsAppError(err))
		return
	}

	if err := h.svc.Send(r.Context(), contactservice.Input{
		Locale:  locale.FromCtx(r.Context()),
		Name:    req.Name,
		Email:   req.Email,
		Subject: req.Subject,
		Message: req.Message,
	}); err != nil {
		api.Error(w, h.logger, api.AsAppError(err))
		return
	}

	api.JSON(w, http.StatusAccepted, SingleResponse{Data: CreateResponse{Status: "accepted"}})
}
