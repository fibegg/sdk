package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fibegg/sdk/internal/contractmanifest"
)

func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "contracts/go-public-api.json", "output file")
	flag.Parse()
	data, err := contractmanifest.Generate(filepath.Join(*root, "fibe"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path := filepath.Join(*root, *out)
	// #nosec G301 -- generated public contract directories are intentionally readable.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// #nosec G306 -- the generated API manifest contains no secrets and is published.
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
