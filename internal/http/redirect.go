package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		http.Error(w, "code parameter is required", http.StatusBadRequest)
		return
	}

	// TODO: lookup long URL from storage and redirect.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}