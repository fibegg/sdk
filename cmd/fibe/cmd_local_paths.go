package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fibegg/sdk/internal/localplaygrounds"
	"github.com/spf13/cobra"
)

type localResolvedPath struct {
	Type       string `json:"type" yaml:"type"`
	Name       string `json:"name" yaml:"name"`
	Path       string `json:"path" yaml:"path"`
	Dir        string `json:"dir" yaml:"dir"`
	LinkDir    string `json:"link_dir" yaml:"link_dir"`
	Playground string `json:"playground,omitempty" yaml:"playground,omitempty"`
	CreatedDir bool   `json:"created_dir,omitempty" yaml:"created_dir,omitempty"`
}

func localPathsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "paths",
		Short: "Resolve local artifact paths without running tools",
		Long: `Resolve local artifact paths for agents and scripts.

The resolver does not run Playwright or capture screenshots. By default it
does not create directories; pass --mkdir when the caller wants the directory
created before writing the artifact.`,
	}
	cmd.AddCommand(localScreenshotPathCmd())
	return cmd
}

func localScreenshotPathCmd() *cobra.Command {
	var name string
	var linkDir string
	var mkdir bool

	cmd := &cobra.Command{
		Use:   "screenshot",
		Short: "Resolve a screenshot output path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("required flag '--name' not set")
			}
			if linkDir == "" {
				linkDir = "/app/playground"
			}
			state, _ := localplaygrounds.LoadCurrentState(linkDir)
			playground := ""
			if state != nil {
				playground = state.Name
			}
			dirParts := []string{linkDir, ".artifacts", "screenshots"}
			if playground != "" {
				dirParts = append(dirParts, safePathSegment(playground))
			}
			dir := filepath.Join(dirParts...)
			if mkdir {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return err
				}
			}
			path := filepath.Join(dir, safePathSegment(name)+".png")
			result := localResolvedPath{
				Type:       "screenshot",
				Name:       name,
				Path:       path,
				Dir:        dir,
				LinkDir:    linkDir,
				Playground: playground,
				CreatedDir: mkdir,
			}
			if effectiveOutput() == "json" || effectiveOutput() == "yaml" {
				output(result)
				return nil
			}
			fmt.Println(path)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Screenshot logical name; sanitized and given a .png extension")
	cmd.Flags().StringVar(&linkDir, "link-dir", "", "Current-link directory (default: /app/playground)")
	cmd.Flags().BoolVar(&mkdir, "mkdir", false, "Create the parent directory")
	return cmd
}

func safePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "artifact"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), ".-")
	if out == "" {
		return "artifact"
	}
	return out
}
