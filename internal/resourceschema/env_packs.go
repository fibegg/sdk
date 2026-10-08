package resourceschema

import (
	"encoding/json"
	"github.com/fibegg/sdk/fibe"
)

func envPackAttachmentSchema() map[string]any {
	return map[string]any{"type": []string{"array", "null"}, "description": "Ordered pack references. Omit or null to preserve/inherit; [] explicitly detaches all packs.", "items": map[string]any{"type": "object", "required": []string{"env_pack_id"}, "additionalProperties": false, "properties": map[string]any{
		"env_pack_id":   map[string]any{"type": "integer", "minimum": 1},
		"service_names": map[string]any{"type": []string{"array", "null"}, "minItems": 1, "uniqueItems": true, "items": map[string]any{"type": "string", "minLength": 1}, "description": "Omitted or null applies to all services; an empty array is invalid."},
	}}}
}
func installEnvPackSchemas(out map[string]map[string]any) {
	out["env_pack"] = map[string]any{
		"create":              paramsSchema[fibe.EnvPackCreateParams]("name", "env"),
		"update":              updateParamsSchemaFor[fibe.EnvPackUpdateParams]("env_pack_id"),
		"attachments_get":     map[string]any{"required": []string{"target", "target_id"}, "properties": map[string]any{"target": map[string]any{"type": "string", "enum": []string{"playgrounds", "specs"}}, "target_id": map[string]any{"type": "integer", "minimum": 1}}},
		"attachments_replace": map[string]any{"required": []string{"target", "target_id"}, "properties": map[string]any{"target": map[string]any{"type": "string", "enum": []string{"playgrounds", "specs"}}, "target_id": map[string]any{"type": "integer", "minimum": 1}, "env_pack_attachments": envPackAttachmentSchema()}},
		"grant_renew":         map[string]any{"required": []string{"target", "target_id", "attachment_id"}, "properties": map[string]any{"target": map[string]any{"type": "string", "enum": []string{"playgrounds", "specs"}}, "target_id": map[string]any{"type": "integer", "minimum": 1}, "attachment_id": map[string]any{"type": "integer", "minimum": 1}}},
	}
	for _, operation := range []string{"attachments_reorder", "attachment_detach", "attachment_retarget"} {
		props := map[string]any{"target": map[string]any{"type": "string", "enum": []string{"playgrounds", "specs"}}, "target_id": map[string]any{"type": "integer", "minimum": 1}}
		required := []string{"target", "target_id"}
		if operation == "attachments_reorder" {
			props["order"] = map[string]any{"type": "array", "items": map[string]any{"type": "integer", "minimum": 1}, "uniqueItems": true}
			required = append(required, "order")
		} else {
			props["env_pack_id"] = map[string]any{"type": "integer", "minimum": 1}
			required = append(required, "env_pack_id")
		}
		if operation == "attachment_retarget" {
			props["service_names"] = map[string]any{"type": []string{"array", "null"}, "minItems": 1, "items": map[string]any{"type": "string", "minLength": 1}, "description": "Omit or null for all services; a present empty list is invalid."}
		}
		out["env_pack"][operation] = map[string]any{"required": required, "properties": props, "additionalProperties": false}
	}
	for _, resource := range []string{"playground", "spec"} {
		for _, operation := range []string{"create", "update"} {
			properties := out[resource][operation].(map[string]any)["properties"].(map[string]any)
			properties["env_pack_attachments"] = envPackAttachmentSchema()
		}
	}
	out["task"]["rerun"].(map[string]any)["properties"].(map[string]any)["env_pack_attachments"] = envPackAttachmentSchema()
	out["task"]["trigger"].(map[string]any)["properties"].(map[string]any)["env_pack_attachments"] = envPackAttachmentSchema()
	out["playground"]["rerun"] = map[string]any{"required": []string{"playground_id"}, "properties": map[string]any{"playground_id": map[string]any{"type": "integer", "minimum": 1}, "name": map[string]any{"type": "string"}, "env_pack_attachments": envPackAttachmentSchema()}}
}

func validateEnvPackOperation(resource, operation string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	switch resource + "." + operation {
	case "env_pack.create":
		var p fibe.EnvPackCreateParams
		if err = json.Unmarshal(data, &p); err == nil {
			err = p.Validate()
		}
	case "env_pack.update":
		var p fibe.EnvPackUpdateParams
		if err = json.Unmarshal(data, &p); err == nil {
			err = p.Validate()
		}
	default:
		if _, ok := payload["env_pack_attachments"]; ok {
			var p fibe.EnvPackAttachmentsParams
			if err = json.Unmarshal(data, &p); err == nil {
				err = p.Validate()
			}
		}
	}
	return err
}

// EnvPackAttachmentSchema describes recipient references for specialized launch tools.
func EnvPackAttachmentSchema() map[string]any { return envPackAttachmentSchema() }

// ValidateEnvPackAttachments validates the source references without resolving authority.
func ValidateEnvPackAttachments(value any) error {
	if value == nil {
		return nil
	}
	return validateValue("env_pack_attachments", value, envPackAttachmentSchema())
}
