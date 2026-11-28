// Package contracts exposes the checked-in REST inventory to the SDK's MCP server.
package contracts

import _ "embed"

//go:embed rest.json
var restJSON string

// REST returns the REST inventory.
func REST() string {
	return restJSON
}
