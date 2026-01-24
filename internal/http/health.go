package http

import "net/http"

// healthzHandler responds with a simple "ok" message for health checks.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// For now always ready. Later: check Redis connectivity, etc.
func readyzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}