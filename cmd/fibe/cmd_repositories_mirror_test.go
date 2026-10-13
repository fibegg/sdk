package main

import (
	"strings"
	"testing"
)

func TestRepositoryMirrorFlagsDescribeBothOwners(t *testing.T) {
	command := repositoryMirrorCmd()
	if err := command.ParseFlags([]string{"--url", "https://github.com/upstream/source", "--name", "destination", "--github-installation-id", "501"}); err != nil {
		t.Fatal(err)
	}
	id, err := command.Flags().GetInt64("github-installation-id")
	if err != nil || id != 501 {
		t.Fatalf("explicit Company installation flag unavailable: %d %v", id, err)
	}
	for _, contract := range []string{"Personal imports use Gitea", "Company imports use the verified GitHub organization", "repositories:read", "repositories:write"} {
		if !strings.Contains(command.Long, contract) {
			t.Fatalf("help omits %q", contract)
		}
	}
}
