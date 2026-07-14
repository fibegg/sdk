//go:build mage

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/sh"
)

var Default = Build

var ldflags = fmt.Sprintf("-s -w -X main.version=%s -X github.com/fibegg/sdk/fibe.sdkVersion=%s -X github.com/fibegg/sdk/internal/mcpserver.Version=%s", version(), version(), version())

func version() string {
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}
	out, _ := sh.Output("git", "describe", "--tags", "--always", "--dirty")
	if out != "" {
		return out
	}
	return "devel"
}

func Build() error {
	fmt.Println("Building fibe...")
	return sh.RunV("go", "build", "-ldflags", ldflags, "-o", "dist/fibe", "./cmd/fibe")
}

func BuildAll() error {
	targets := []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
	}

	for _, t := range targets {
		ext := ""
		if t.goos == "windows" {
			ext = ".exe"
		}
		out := fmt.Sprintf("dist/fibe-%s-%s%s", t.goos, t.goarch, ext)
		fmt.Printf("Building %s...\n", out)
		env := map[string]string{"GOOS": t.goos, "GOARCH": t.goarch, "CGO_ENABLED": "0"}
		if err := sh.RunWith(env, "go", "build", "-ldflags", ldflags, "-o", out, "./cmd/fibe"); err != nil {
			return err
		}
	}
	return nil
}

func Test() error {
	if err := ToolsDocsCheck(); err != nil {
		return err
	}
	return sh.RunV("go", "tool", "gotestsum", "--format", "testname", "--", "./fibe/...", "./fibetest/...", "./internal/...", "./cmd/fibe/...", "-count=1", "-timeout", "120s", "-short")
}

func ToolsDocs() error {
	return sh.RunV("go", "run", "./scripts/docs")
}

func ToolsDocsCheck() error {
	return sh.RunV("go", "run", "./scripts/docs", "--check")
}

func IntegrationTest() error {
	return sh.RunV("go", "tool", "gotestsum", "--format", "testname", "--", "-tags=integration", "./integration/...", "./internal/mcpserver/...", "-count=1", "-timeout", "600s", "-parallel", "8")
}

// ChatE2E runs provider chat runtime E2E tests.
func ChatE2E() error {
	return sh.RunV(
		"go", "tool", "gotestsum",
		"--format", "testname",
		"--",
		"./integration/...",
		"-run", "TestAgentRuntimeMatrix",
		"-count=1",
		"-timeout", "1800s",
		"-parallel", "1",
	)
}

// ChatE2EHelp prints provider chat E2E env vars without running tests.
func ChatE2EHelp() {
	fmt.Print(chatE2EHelpText())
}

func Lint() error {
	if err := sh.RunV("go", "vet", "./..."); err != nil {
		return err
	}
	if err := sh.RunV("go", "tool", "staticcheck", "./..."); err != nil {
		return err
	}
	if err := sh.RunV("go", "tool", "revive", "-config", ".revive.toml", "./..."); err != nil {
		return err
	}
	return nil
}

// Check runs the same hermetic quality gate used by CI and releases.
func Check() error {
	steps := []func() error{
		FormatCheck,
		func() error { return sh.RunV("go", "mod", "verify") },
		func() error { return sh.RunV("go", "mod", "tidy", "-diff") },
		Lint,
		Security,
		ToolsDocsCheck,
		func() error { return sh.RunV("go", "build", "./examples/...") },
		func() error {
			return sh.RunV("go", "tool", "gotestsum", "--format", "testname", "--",
				"./fibe/...", "./fibetest/...", "./internal/...", "./cmd/fibe/...",
				"-race", "-shuffle=on", "-count=1", "-timeout", "180s", "-short")
		},
		BuildAll,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// Security runs pinned vulnerability and source-security scanners.
func Security() error {
	if err := sh.RunV("go", "tool", "govulncheck", "./..."); err != nil {
		return err
	}
	return sh.RunV("go", "tool", "gosec", "-quiet", "./...")
}

// FormatCheck rejects Go files whose tracked formatting differs from gofmt.
func FormatCheck() error {
	var files []string
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (path == ".git" || path == "dist" || path == "graphify-out") {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	args := append([]string{"-l"}, files...)
	out, err := sh.Output("gofmt", args...)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Errorf("gofmt required for:\n%s", out)
	}
	return nil
}

func Clean() error {
	return os.RemoveAll("dist")
}

func Install() error {
	fmt.Printf("Installing to %s/bin/fibe...\n", gopath())
	return sh.RunV("go", "install", "-ldflags", ldflags, "./cmd/fibe")
}

func gopath() string {
	if gp := os.Getenv("GOPATH"); gp != "" {
		return gp
	}
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return home + "\\go"
	}
	return home + "/go"
}

func chatE2EHelpText() string {
	rows := [][]string{
		{"Gemini OAuth", "GEMINI_OAUTH_JSON", "gemini", "pro"},
		{"Gemini API key", "GEMINI_API_KEY", "gemini", "gemini-2.5-flash-lite"},
		{"Claude manual", "CLAUDE_CODE_OAUTH_TOKEN", "claude-code", "haiku"},
		{"Claude API key", "ANTHROPIC_API_KEY", "claude-code", "haiku"},
		{"Codex auth JSON", "CODEX_AUTH_JSON", "openai-codex", "gpt-5.4-mini"},
		{"Codex API key", "OPENAI_API_KEY", "openai-codex", "gpt-5.4-mini"},
		{"Cursor API key", "CURSOR_API_KEY", "cursor", "default"},
		{"OpenCode OpenRouter", "OPENROUTER_API_KEY", "opencode", "google/gemini-2.5-flash-lite"},
		{"OpenCode Anthropic", "ANTHROPIC_API_KEY", "opencode", "anthropic/claude-haiku-4-5"},
		{"OpenCode OpenAI", "OPENAI_API_KEY", "opencode", "openai/gpt-5-mini"},
		{"OpenCode Gemini", "GEMINI_API_KEY", "opencode", "google/gemini-2.5-flash-lite"},
	}

	var b strings.Builder
	b.WriteString(`chatE2E runs TestAgentRuntimeMatrix only.

Required for any runnable row:
  FIBE_API_KEY
  FIBE_DOMAIN optional; defaults to localhost:3000
  FIBE_TEST_MARQUEE_ID

Prompt:
  MESSAGE or FIBE_TEST_AGENT_MESSAGE overrides the first chat message.
  CHAT_E2E_FOLLOWUPS or FIBE_TEST_AGENT_FOLLOWUPS controls extra short prompts (default 5).
  CHAT_E2E_MIN_ENTRIES or FIBE_TEST_AGENT_MIN_ENTRIES controls the messages/activity threshold (default 5, test requires count > threshold).

Filtering:
  CHAT_E2E_CASE filters by case/provider/model/env substring.

Rows:
`)
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("  %-20s credential=%-20s provider=%-13s model=%s\n", row[0], row[1], row[2], row[3]))
	}
	b.WriteString(`
Examples:
  OPENAI_API_KEY=SECRET MESSAGE="[SYSCHECK]" mage chatE2E
  OPENAI_API_KEY=SECRET MESSAGE="[SYSCHECK]" CHAT_E2E_FOLLOWUPS=0 CHAT_E2E_MIN_ENTRIES=0 mage chatE2E
  CHAT_E2E_CASE=opencode_anthropic ANTHROPIC_API_KEY=SECRET MESSAGE="hello" mage chatE2E
  mage chatE2EHelp
`)
	return b.String()
}
