package restcontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRESTContractIncludesDirectClientEndpointsWithoutConfigurationHelpers(t *testing.T) {
	dir := t.TempDir()
	source := `package fibe
import("context";"net/http";"fmt")
type Client struct{}
func(c *Client) BaseURL() string{return "local"}
func(c *Client) Recover(ctx context.Context, id int64)(any,error){var result any;err:=c.do(ctx,http.MethodPost,fmt.Sprintf("/api/principals/%d/recover",id),nil,&result);return result,err}
`
	if err := os.WriteFile(filepath.Join(dir, "client.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	var contract Contract
	if err = json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	if len(contract.Operations) != 1 {
		t.Fatalf("operations %+v", contract.Operations)
	}
	op := contract.Operations[0]
	if op.Name != "Client.Recover" || op.HTTPMethod != "POST" || op.PathTemplate != "/api/principals/{id}/recover" {
		t.Fatalf("endpoint lost: %+v", op)
	}
}
