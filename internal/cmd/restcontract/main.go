// Command restcontract generates the portable SDK REST contract.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
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
	write(filepath.Join(*root, *out), data)
	if path.Clean(*out) == "contracts/rest.json" {
		var compact bytes.Buffer
		if err := json.Compact(&compact, data); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		write(filepath.Join(*root, "internal", "restcontract", "embedded", "rest.json"), append(compact.Bytes(), '\n'))
	}
}

func write(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { // #nosec G301 -- generated contracts are readable.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil { // #nosec G306 -- generated contracts contain no secrets.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
