package integration

import (
	"github.com/fibegg/sdk/internal/testedition"
	"testing"
)

func requireEdition(t *testing.T) { t.Helper(); testedition.Require(t, "integration") }
func communityEdition() bool      { return testedition.Current() == testedition.Community }
func runEdition(t *testing.T, name string, body func(*testing.T)) bool {
	t.Helper()
	return testedition.Run(t, "integration", name, body)
}
