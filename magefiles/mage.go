//go:build mage

package main

import (
	"fmt"
	"os"
	"runtime"

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
	return sh.RunV("go", "run", "gotest.tools/gotestsum@latest", "--format", "testname", "--", "./integration/...", "-v", "-count=1", "-timeout", "600s", "-parallel", "8")
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
