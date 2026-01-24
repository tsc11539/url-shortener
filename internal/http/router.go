package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures a new HTTP router.
func NewRouter() http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	// Health check endpoint
	r.Get("/healthz", healthzHandler)
	r.Get("/readyz", readyzHandler)

	// Redirect endpoint (URL shortener core)
	r.Get("/{code}", redirectHandler)

	return r
}