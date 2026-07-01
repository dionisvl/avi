package api

import "github.com/go-chi/chi/v5"

// Handler is the interface for all API handlers
type Handler interface {
	Routes() chi.Router
}
