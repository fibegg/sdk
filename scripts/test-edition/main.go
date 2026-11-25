// Command test-edition evaluates the reviewed live SDK integration selection offline.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/fibegg/sdk/internal/testedition"
)

func main() {
	edition := flag.String("edition", os.Getenv(testedition.Environment), "community, enterprise or full (legacy default)")
	pkg := flag.String("package", "", "integration or internal/mcpserver; empty selects both")
	run := flag.String("run", ".", "Go -run expression")
	list := flag.Bool("list", false, "print selected package/case names without contacting an API")
	jsonOut := flag.Bool("json", false, "print machine-readable evaluation")
	review := flag.String("fingerprint", "", "print canonical digest of one reviewed dynamic package/TestName")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected positional arguments"))
	}
	mode, e := testedition.Mode(*edition)
	if e != nil {
		fail(e)
	}
	if *review != "" {
		root, e := testedition.Root()
		if e != nil {
			fail(e)
		}
		if *review == "*" {
			digests, e := testedition.DynamicReviews(root)
			if e != nil {
				fail(e)
			}
			if e = json.NewEncoder(os.Stdout).Encode(digests); e != nil {
				fail(e)
			}
			return
		}
		digest, e := testedition.Fingerprint(root, *review)
		if e != nil {
			fail(e)
		}
		fmt.Println(digest)
		return
	}
	m, e := testedition.Load()
	if e != nil {
		fail(e)
	}
	root, e := testedition.Root()
	if e != nil {
		fail(e)
	}
	if e = testedition.Validate(root, m); e != nil {
		fail(e)
	}
	cases, e := m.Select(mode, *pkg, *run)
	if e != nil {
		fail(e)
	}
	if *jsonOut {
		if e = json.NewEncoder(os.Stdout).Encode(struct {
			Edition string             `json:"edition"`
			Count   int                `json:"count"`
			Cases   []testedition.Case `json:"cases"`
		}{mode, len(cases), cases}); e != nil {
			fail(e)
		}
		return
	}
	if !*list {
		fmt.Printf("edition=%s selected=%d\n", mode, len(cases))
	}
	for _, c := range cases {
		fmt.Printf("%s %s\n", c.Package, c.Name)
	}
}
func fail(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(2) }
