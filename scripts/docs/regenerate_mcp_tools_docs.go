package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fibegg/sdk/internal/mcpserver"
)

func main() {
	check := flag.Bool("check", false, "exit non-zero if generated docs differ from files on disk")
	outDir := flag.String("out-dir", ".", "repository root containing the generated markdown files")
	flag.Parse()

	srv := mcpserver.New(mcpserver.DefaultConfig())
	if err := srv.RegisterAll(); err != nil {
		fatalf("register MCP tools: %v", err)
	}

	tools := srv.AllTools()
	docs := mcpserver.GenerateToolDocs(tools)
	files := map[string]string{
		"fibe_mcp_tools_catalog.md": docs.CatalogMarkdown,
		"fibe_tools_table.md":       docs.TableMarkdown,
	}

	var drift []string
	for name, body := range files {
		path := filepath.Join(*outDir, name)
		if *check {
			current, err := os.ReadFile(path)
			if err != nil {
				fatalf("read %s: %v", path, err)
			}
			if string(current) != body {
				drift = append(drift, name)
			}
			continue
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			fatalf("write %s: %v", path, err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", path)
	}

	if len(drift) > 0 {
		fatalf("MCP tool docs are stale: %s", strings.Join(drift, ", "))
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
