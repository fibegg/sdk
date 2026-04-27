//go:build mage

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/magefile/mage/sh"
)

var Default = Build

var ldflags = fmt.Sprintf("-s -w -X main.version=%s", version())

func version() string {
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}
	out, _ := sh.Output("git", "describe", "--tags", "--always", "--dirty")
	if out != "" {
		return out
	}
	return "dev"
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
	return sh.RunV("go", "run", "gotest.tools/gotestsum@latest", "--format", "testname", "--", "./fibe/...", "-count=1", "-timeout", "30s")
}

func IntegrationTest() error {
	return sh.RunV("go", "run", "gotest.tools/gotestsum@latest", "--format", "testname", "--", "./integration/...", "./internal/mcpserver/...", "-count=1", "-timeout", "600s", "-parallel", "8")
}

// ChatE2E runs provider chat runtime E2E tests.
func ChatE2E() error {
	return sh.RunV(
		"go", "run", "gotest.tools/gotestsum@latest",
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
	return sh.RunV("go", "vet", "./...")
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
		{"Gemini OAuth", "FIBE_TEST_AGENT_GEMINI_OAUTH_JSON", "GEMINI_OAUTH_JSON", "gemini", "pro"},
		{"Gemini API key", "FIBE_TEST_AGENT_GEMINI_API_KEY", "GEMINI_KEY", "gemini", "flash-lite"},
		{"Claude manual", "FIBE_TEST_AGENT_CLAUDE_CODE_OAUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN", "claude-code", "haiku"},
		{"Claude API key", "FIBE_TEST_AGENT_ANTHROPIC_API_KEY", "ANTHROPIC_KEY", "claude-code", "haiku"},
		{"Codex auth JSON", "FIBE_TEST_AGENT_CODEX_AUTH_JSON", "CODEX_AUTH_JSON", "openai-codex", "gpt-5.4-mini"},
		{"Codex API key", "FIBE_TEST_AGENT_OPENAI_API_KEY", "OPENAI_KEY", "openai-codex", "gpt-5.4-mini"},
		{"Cursor API key", "FIBE_TEST_AGENT_CURSOR_API_KEY", "CURSOR_KEY", "cursor", "default"},
		{"OpenCode OpenRouter", "FIBE_TEST_AGENT_OPENCODE_OPENROUTER_API_KEY", "OPENCODE_OPENROUTER_KEY", "opencode", "deepseek/deepseek-chat-v3.1"},
		{"OpenCode Anthropic", "FIBE_TEST_AGENT_OPENCODE_ANTHROPIC_API_KEY", "OPENCODE_ANTHROPIC_KEY", "opencode", "anthropic/claude-sonnet-4"},
		{"OpenCode OpenAI", "FIBE_TEST_AGENT_OPENCODE_OPENAI_API_KEY", "OPENCODE_OPENAI_KEY", "opencode", "openai/gpt-4.1"},
		{"OpenCode Gemini", "FIBE_TEST_AGENT_OPENCODE_GEMINI_API_KEY", "OPENCODE_GEMINI_KEY", "opencode", "google/gemini-2.5-pro"},
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
		b.WriteString(fmt.Sprintf("  %-20s primary=%-45s alias=%-24s provider=%-13s model=%s\n", row[0], row[1], row[2], row[3], row[4]))
	}
	b.WriteString(`
Examples:
  OPENAI_KEY=SECRET MESSAGE="[SYSCHECK]" mage chatE2E
  OPENAI_KEY=SECRET MESSAGE="[SYSCHECK]" CHAT_E2E_FOLLOWUPS=0 CHAT_E2E_MIN_ENTRIES=0 mage chatE2E
  CHAT_E2E_CASE=opencode_openai OPENCODE_OPENAI_KEY=SECRET MESSAGE="hello" mage chatE2E
  mage chatE2EHelp
`)
	return b.String()
}
