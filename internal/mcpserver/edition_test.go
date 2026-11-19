package mcpserver

import (
	"fmt"
	"github.com/fibegg/sdk/internal/testedition"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	listing, err := testedition.Prepare("internal/mcpserver", true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if listing && testedition.Current() == testedition.Community {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func runEdition(t *testing.T, name string, body func(*testing.T)) bool {
	t.Helper()
	return testedition.Run(t, "internal/mcpserver", name, body)
}
