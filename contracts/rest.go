// Package contracts exposes the checked-in REST inventory to the SDK's MCP server.
package contracts

import _ "embed"

//go:embed rest.json
var restJSON []byte

// REST returns the REST inventory.
func REST() []byte {
	return restJSON
}
