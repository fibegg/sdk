package mcpserver

import (
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestPlaygroundWaitProgressMessageIncludesServiceReadiness(t *testing.T) {
	msg := playgroundWaitProgressMessage(&fibe.PlaygroundStatus{
		Status: "running",
		Services: []fibe.PlaygroundServiceInfo{
			{Name: "web", Status: "running", Health: "healthy", Running: true},
			{Name: "worker", Status: "created", Running: false},
		},
	})

	if !strings.Contains(msg, "status: running") {
		t.Fatalf("missing status in progress message: %s", msg)
	}
	if !strings.Contains(msg, "services: web=running/health=healthy/running, worker=created/not-running") {
		t.Fatalf("missing service readiness in progress message: %s", msg)
	}
}
