package mcpserver

import (
	"fmt"
	"os"
	"testing"

	"github.com/fibegg/sdk/internal/testedition"
)

func TestMain(m *testing.M) {
	listing, err := testedition.Prepare("internal/mcpserver", true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if listing && testedition.Current() != testedition.Full {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func runEdition(t *testing.T, name string, body func(*testing.T)) bool {
	t.Helper()
	return testedition.Run(t, "internal/mcpserver", name, body)
}
