package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
)

func main() {
	admin := fibe.NewClient(fibe.WithMaxRetries(2))
	fmt.Printf("admin Default config timeout: %v\n", admin.HTTPClientTimeout())
}
