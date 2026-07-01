package health

import (
	"encoding/json"
	"net/http"
)

// HealthResponse represents the health check response
// @Description Health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type Handler struct {
	version string
}

func NewHandler(version string) *Handler {
	return &Handler{version: version}
}

// ServeHTTP handles health check requests
// @Summary Health check
// @Description Get API health status
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:  "ok",
		Version: h.version,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
