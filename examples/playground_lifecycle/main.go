package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func main() {
	key := os.Getenv("FIBE_API_KEY")
	if key == "" {
		log.Fatal("FIBE_API_KEY must be set")
	}

	client := fibe.NewClient(fibe.WithAPIKey(key))
	ctx := context.Background()

	fmt.Println("Launching Playground...")

	ctxWithIdemp := fibe.WithIdempotencyKey(ctx, fibe.NewIdempotencyKey())

	pg, err := client.Playgrounds.Create(ctxWithIdemp, &fibe.PlaygroundCreateParams{
		Name:   "example-python-lifecycle",
		SpecID: 1,
	})
	if err != nil {
		log.Fatalf("Failed to create playground: %v", err)
	}

	fmt.Printf("Created playground %d (Status: %s)\n", pg.ID, pg.Status)

	for {
		fetched, err := client.Playgrounds.Get(ctx, pg.ID)
		if err != nil {
			log.Fatalf("Failed to fetch playground: %v", err)
		}

		fmt.Printf("Current status: %s\n", fetched.Status)
		if fetched.Status == "running" {
			fmt.Println("Playground is fully available!")
			break
		} else if fetched.Status == "failed" {
			log.Fatal("Playground failed to start.")
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Println("\nCreating an interactive agent...")
	ag, err := client.Agents.Create(ctx, &fibe.AgentCreateParams{
		Name:     "sys-operator",
		Provider: "gemini",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	msgResp, err := client.Agents.Chat(ctx, ag.ID, &fibe.AgentChatParams{
		Text: "Can you reach the Playground?",
	})
	if err != nil {
		log.Fatalf("Chat attempt failed: %v", err)
	}
	fmt.Printf("Agent acknowledged: %v\n", msgResp)

	fmt.Println("\nDeleting Playground...")
	if err := client.Playgrounds.Delete(ctx, pg.ID); err != nil {
		log.Printf("Deletion failed: %v", err)
	} else {
		fmt.Println("Playground deleted.")
	}
}
