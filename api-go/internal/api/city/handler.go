package city

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dionisvl/avi/api-go/internal/api"
	itemapi "github.com/dionisvl/avi/api-go/internal/api/item"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
	"github.com/dionisvl/avi/api-go/internal/repository/city"
)

type Handler struct {
	cityRepo city.Repository
	logger   *slog.Logger
}

func NewHandler(cityRepo city.Repository, logger *slog.Logger) *Handler {
	return &Handler{cityRepo: cityRepo, logger: logger}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	return r
}

type ListResponse struct {
	Data []itemapi.CityResponse `json:"data"`
}

// @Summary List all cities
// @Tags cities
// @Produce json
// @Success 200 {object} ListResponse
// @Router /api/v1/cities [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	cities, err := h.cityRepo.List(r.Context())
	if err != nil {
		api.Error(w, h.logger, apierr.New(apierr.ErrInternal, "failed to list cities"))
		return
	}

	items := make([]itemapi.CityResponse, 0, len(cities))
	for _, c := range cities {
		items = append(items, itemapi.MapCityToResponse(c))
	}

	api.JSON(w, http.StatusOK, ListResponse{Data: items})
}
