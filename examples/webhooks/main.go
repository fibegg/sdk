package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/fibegg/sdk/fibe"
)

func main() {
	secret := "your-webhook-secret"

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

	fmt.Println("Webhook server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
