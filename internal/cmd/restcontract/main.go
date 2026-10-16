// Command restcontract generates the portable SDK REST contract.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fibegg/sdk/internal/restcontract"
)

func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "contracts/rest.json", "output file")
	flag.Parse()
	data, err := restcontract.Generate(filepath.Join(*root, "fibe"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path := filepath.Join(*root, *out)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { // #nosec G301 -- generated public contracts are readable.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil { // #nosec G306 -- generated public contracts contain no secrets.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
