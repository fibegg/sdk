package fibe

import (
	"reflect"
	"strings"
	"testing"
)

func TestAgentTypesExposeRuntimeSettings(t *testing.T) {
	assertJSONTags(t, reflect.TypeOf(Agent{}), []string{
		"prompt",
		"mcp_json",
		"post_init_script",
		"custom_env",
		"cli_version",
		"provider_args",
		"skill_toggles",
		"effective_prompt",
		"effective_model_options",
		"effective_memory_limit",
		"effective_cpu_limit",
		"effective_cli_version",
	})
}

func TestAgentCreateParamsExposeRuntimeSettings(t *testing.T) {
	assertJSONTags(t, reflect.TypeOf(AgentCreateParams{}), []string{
		"prompt",
		"mcp_json",
		"post_init_script",
		"custom_env",
		"cli_version",
		"provider_args",
		"skill_toggles",
	})
}

func TestAgentUpdateParamsExposeRuntimeSettings(t *testing.T) {
	assertJSONTags(t, reflect.TypeOf(AgentUpdateParams{}), []string{
		"prompt",
		"mcp_json",
		"post_init_script",
		"custom_env",
		"cli_version",
		"provider_args",
		"skill_toggles",
	})
}

func assertJSONTags(t *testing.T, typ reflect.Type, want []string) {
	t.Helper()

	got := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name != "" && name != "-" {
			got[name] = true
		}
	}

	for _, tag := range want {
		if !got[tag] {
			t.Fatalf("%s missing json tag %q; got %#v", typ.Name(), tag, got)
		}
	}
}
