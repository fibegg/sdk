package resourceschema

import "testing"

func TestEnvPackSchemasKeepAbsentNullAndEmptyDistinct(t *testing.T) {
	for _, value := range []any{nil, []any{}} {
		p := map[string]any{"target": "playgrounds", "target_id": 7, "env_pack_attachments": value}
		if _, _, err := ValidateMutationPayload("env_packs", "attachments_replace", p); err != nil {
			t.Fatal(err)
		}
	}
	for _, names := range []any{nil, []any{"web"}} {
		row := map[string]any{"env_pack_id": 1, "service_names": names}
		if _, _, err := ValidateMutationPayload("env_pack", "attachments_replace", map[string]any{"target": "specs", "target_id": 3, "env_pack_attachments": []any{row}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []map[string]any{{"env_pack_id": 1, "service_names": []any{}}, {"env_pack_id": 1, "grantor_id": 2}, {"env_pack_id": 1, "service_names": []any{"web", "web"}}} {
		if _, _, err := ValidateMutationPayload("env_pack", "attachments_replace", map[string]any{"target": "specs", "target_id": 3, "env_pack_attachments": []any{row}}); err == nil {
			t.Fatalf("invalid attachment accepted: %v", row)
		}
	}
	for _, env := range []map[string]any{{"EMPTY": "", "TEXT": "line\n\"quote\""}} {
		if _, _, err := ValidateMutationPayload("env_pack", "create", map[string]any{"name": "readable", "env": env}); err != nil {
			t.Fatal(err)
		}
	}
	for _, env := range []map[string]any{{"BAD-KEY": "x"}, {"NESTED": map[string]any{"value": "x"}}} {
		if _, _, err := ValidateMutationPayload("env_pack", "create", map[string]any{"name": "invalid", "env": env}); err == nil {
			t.Fatal("invalid pack values accepted")
		}
	}
}

func TestEnvPackLaunchSchemaAdmitsOnlyRecipientReferences(t *testing.T) {
	for _, selection := range []any{nil, []any{}, []any{map[string]any{"env_pack_id": 7, "service_names": []any{"web"}}}} {
		if err := ValidateEnvPackAttachments(selection); err != nil {
			t.Fatal(err)
		}
	}
	for _, selection := range []any{[]any{map[string]any{"env_pack_id": 7, "grantor_id": 99}}, []any{map[string]any{"env_pack_id": 7, "service_names": []any{}}}} {
		if err := ValidateEnvPackAttachments(selection); err == nil {
			t.Fatal("invalid recipient reference accepted")
		}
	}
}
