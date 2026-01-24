package main

import (
	"log"
	"net/http"
	"os"
	"time"

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

	log.Printf("Starting server on port %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("Server Ended")
}
