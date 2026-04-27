//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fibegg/sdk/fibe"
)

func main() {
	client := fibe.NewClient(fibe.WithAPIKey(os.Getenv("FIBE_API_KEY")), fibe.WithDomain(os.Getenv("FIBE_DOMAIN")))

	res, err := client.ImportTemplates.List(context.Background(), &fibe.ImportTemplateListParams{})
	if err != nil {
		panic(err)
	}
	for _, t := range res.Data {
		fmt.Printf("Template ID: %d, Name: %s\n", t.ID, t.Name)
	}
}
