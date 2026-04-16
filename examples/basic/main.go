package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fibegg/sdk/fibe"
)

func main() {
	client := fibe.NewClient(
		fibe.WithAPIKey("your-api-key"),
		fibe.WithCircuitBreaker(fibe.DefaultBreakerConfig),
		fibe.WithRateLimitAutoWait(),
		fibe.WithDebug(),
	)

	ctx := context.Background()

	player, err := client.APIKeys.Me(ctx)
	if err != nil {
		log.Fatalf("auth failed: %v", err)
	}
	fmt.Printf("Authenticated as: %s\n", player.Username)

	playgrounds, err := client.Playgrounds.List(ctx, nil)
	if err != nil {
		log.Fatalf("list playgrounds: %v", err)
	}
	for _, pg := range playgrounds.Data {
		fmt.Printf("  [%d] %s — %s\n", pg.ID, pg.Name, pg.Status)
	}

	agents, err := client.Agents.List(ctx, nil)
	if err != nil {
		log.Fatalf("list agents: %v", err)
	}
	for _, ag := range agents.Data {
		fmt.Printf("  [%d] %s — %s (auth: %v)\n", ag.ID, ag.Name, ag.Provider, ag.Authenticated)
	}
}
