package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func main() {
	secret := os.Getenv("FIBE_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("FIBE_WEBHOOK_SECRET is required")
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		payload, err := fibe.VerifyWebhookSignature(r, secret)
		if err != nil {
			log.Printf("invalid webhook: %v", err)
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		fmt.Printf("Event: %s at %s\n", payload.Event, payload.Timestamp)
		fmt.Printf("Data: %v\n", payload.Data)

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("webhook server shutdown: %v", err)
		}
	}()
	fmt.Println("Webhook server listening on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
