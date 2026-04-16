package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/fibetest"
)

func main() {
	// 1. Boot the built-in Fibe Mock Server
	// This spins up a localized httptest.Server on your machine containing
	// pre-wired routes and handlers simulating the Fibe Cloud ecosystem.
	mockServer := fibetest.NewMockServer()
	defer mockServer.Close()

	fmt.Printf("Mock Fibe API server running at: %s\n", mockServer.URL())

	// 2. Point the standard Fibe Client to your Mock Domain
	// Provide any string for the API key since authorization is bypassed locally 
	client := fibe.NewClient(
		fibe.WithAPIKey("pk_test_mocked_env"),
		fibe.WithDomain(mockServer.Domain()),
	)

	ctx := context.Background()

	// 3. Execute typical API logic securely and immediately in your CI/CD pipelines
	// without traversing the public internet.
	player, err := client.APIKeys.Me(ctx)
	if err != nil {
		log.Fatalf("Failed to fetch Me profile: %v", err)
	}

	fmt.Printf("Authenticated Mock User: %s (ID: %d)\n", player.Username, player.ID)

	pgs, err := client.Playgrounds.List(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to retrieve playgrounds: %v", err)
	}

	fmt.Printf("Locally mocked playgrounds retrieved: %d\n", len(pgs.Data))
	for _, pg := range pgs.Data {
		fmt.Printf("- %s (Status: %s)\n", pg.Name, pg.Status)
	}
}
