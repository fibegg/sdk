package fibe

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGreenfieldLinkedPathCheckoutMetadataRoundTrip(t *testing.T) {
	original := GreenfieldLinkedPath{
		Name:    "app",
		Path:    "/app/playground/app",
		Target:  "/opt/fibe/playgrounds/demo/props/app/main",
		Service: "app",
		Prop:    "app",
		Branch:  "main",
	}.WithCheckoutMetadata("github.com/acme/app", []string{"app", "setup"})

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored GreenfieldLinkedPath
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.CheckoutRepository() != "github.com/acme/app" {
		t.Fatalf("repository=%q", restored.CheckoutRepository())
	}
	if !reflect.DeepEqual(restored.CheckoutServices(), []string{"app", "setup"}) {
		t.Fatalf("services=%v", restored.CheckoutServices())
	}
}
