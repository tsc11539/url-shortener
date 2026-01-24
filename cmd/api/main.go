package main

import (
	"log"
	"net/http"
	"os"
	"time"
	"context"
	"os/signal"
	"syscall"
	"errors"

	httpx "github.com/tsc11539/url-shortener/internal/http"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      httpx.NewRouter(),
		ReadTimeout:  5 * time.Second,
	}

	// Run server in background.
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Starting server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Listen for termination signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("Received signal %s, shutting down...", sig)
	case err := <-errCh:
		if err != nil {
			log.Printf("Server error: %v", err)
			os.Exit(1)
		}

		// server closed normally
		return
	}

	// Graceful shutdown with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		os.Exit(1)
	}

	log.Println("Server stopped gracefully")
}
