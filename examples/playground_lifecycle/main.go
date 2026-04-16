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

	// 1. Create a new Playground
	fmt.Println("Launching robust Python playground...")
	
	// Optional: You can enforce reliable network guarantees with an Idempotency-Key
	ctxWithIdemp := fibe.WithIdempotencyKey(ctx, fibe.NewIdempotencyKey())

	pg, err := client.Playgrounds.Create(ctxWithIdemp, &fibe.PlaygroundCreateParams{
		Name:       "example-python-lifecycle",
		PlayspecID: 1,
	})
	if err != nil {
		log.Fatalf("Failed to create playground: %v", err)
	}

	fmt.Printf("Created playground %d (Status: %s)\n", pg.ID, pg.Status)

	// 2. Await Running Status utilizing robust retries
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

	// 3. Issue commands via Mutters (Interactive Agent interaction)
	fmt.Println("\nCreating an interactive agent...")
	ag, err := client.Agents.Create(ctx, &fibe.AgentCreateParams{
		Name:     "sys-operator",
		Provider: "gemini", 
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	msgResp, err := client.Agents.Chat(ctx, ag.ID, &fibe.AgentChatParams{
		Text: "Hello! Are you successfully wired into the Python playground?",
	})
	if err != nil {
		log.Fatalf("Chat attempt failed: %v", err)
	}
	fmt.Printf("Agent acknowledged: %v\n", msgResp)

	// 4. Secure Cleanup
	fmt.Println("\nCleaning up infrastructure...")
	if err := client.Playgrounds.Delete(ctx, pg.ID); err != nil {
		log.Printf("Deletion encountered an issue: %v", err)
	} else {
		fmt.Println("Playground safely deleted.")
	}
}
