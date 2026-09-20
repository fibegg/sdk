package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/fibetest"
)

func main() {
	mockServer := fibetest.NewMockServer()
	defer mockServer.Close()

	fmt.Printf("Mock Fibe API server running at: %s\n", mockServer.URL())

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
