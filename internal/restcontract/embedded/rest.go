// Package embedded exposes the generated REST inventory to the SDK's MCP server.
package embedded

import _ "embed"

//go:embed rest.json
var restJSON string

// REST returns the REST inventory.
func REST() string {
	return restJSON
}
