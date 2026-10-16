package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerTeamTools() {
	id := func(description string) map[string]any {
		return map[string]any{"type": "integer", "minimum": 1, "description": description}
	}
	add := func(name, description string, read, destructive bool, fields map[string]any, required []string, handler func(context.Context, *fibe.Client, int64, map[string]any) (any, error)) {
		fields["team_id"] = id("Explicit Team owner expectation; never grants authority")
		required = append([]string{"team_id"}, required...)
		if destructive {
			fields["confirm"] = map[string]any{"type": "boolean", "description": "Confirm this operation"}
		}
		schema := map[string]any{"type": "object", "properties": fields, "required": required, "additionalProperties": false}
		s.addTool(&toolImpl{name: name, description: "[MODE:DIALOG] " + description, tier: tierOther, annotations: toolAnnotations{ReadOnly: read, Destructive: destructive, Idempotent: read}, handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			if _, removed := args["company_id"]; removed {
				return nil, fmt.Errorf("FIBE field %q was removed; use %q", "company_id", "team_id")
			}
			for field := range args {
				if _, supported := fields[field]; !supported {
					return nil, fmt.Errorf("unsupported Team operation field %q", field)
				}
			}
			team, err := requiredPositiveID(args, "team_id")
			if err != nil {
				return nil, err
			}
			return handler(ctx, c, team, args)
		}}, mcp.NewTool(name, withRawInputSchema(schema)))
	}
	add("fibe_team_principals_list", "List Team service principals.", true, false, map[string]any{}, nil, func(ctx context.Context, c *fibe.Client, team int64, _ map[string]any) (any, error) {
		return c.TeamServicePrincipals(ctx, team)
	})
	add("fibe_team_principal_create", "Issue a Team service principal with live human admin authority.", false, false, map[string]any{"name": map[string]any{"type": "string", "minLength": 1}}, []string{"name"}, func(ctx context.Context, c *fibe.Client, team int64, args map[string]any) (any, error) {
		return c.CreateTeamServicePrincipal(ctx, team, argString(args, "name"))
	})
	add("fibe_team_principal_revoke", "Revoke a Team service principal with live human admin authority.", false, true, map[string]any{"principal_id": id("Service principal ID")}, []string{"principal_id"}, func(ctx context.Context, c *fibe.Client, team int64, args map[string]any) (any, error) {
		principal, err := requiredPositiveID(args, "principal_id")
		if err != nil {
			return nil, err
		}
		return nil, c.RevokeTeamServicePrincipal(ctx, team, principal)
	})
	add("fibe_team_runtime_principal_recover", "Recover a revoked runtime principal using a live human Team admin. Explicit [] creates a replacement without rebinding agents. Completed identical selections replay; changed selections conflict. Returns adoption states, never runtime key IDs or tokens.", false, true, map[string]any{"principal_id": id("Revoked runtime principal ID"), "replacement_agent_ids": map[string]any{"type": "array", "uniqueItems": true, "items": id("Agent ID")}}, []string{"principal_id", "replacement_agent_ids"}, func(ctx context.Context, c *fibe.Client, team int64, args map[string]any) (any, error) {
		principal, err := requiredPositiveID(args, "principal_id")
		if err != nil {
			return nil, err
		}
		var selection struct {
			IDs []int64 `json:"replacement_agent_ids"`
		}
		encoded, encodeErr := json.Marshal(args["replacement_agent_ids"])
		if encodeErr != nil {
			return nil, fmt.Errorf("replacement_agent_ids must be an explicit list of integer Agent IDs")
		}
		if err = json.Unmarshal(encoded, &selection.IDs); err != nil {
			return nil, err
		}
		return c.RecoverRuntimePrincipal(ctx, team, principal, selection.IDs)
	})
	add("fibe_team_subscriptions_list", "Read current subscriptions and durable cancellation states, including failed/authority-lost receipts.", true, false, map[string]any{}, nil, func(ctx context.Context, c *fibe.Client, team int64, _ map[string]any) (any, error) {
		return c.TeamSubscriptions(ctx, team)
	})
	add("fibe_team_subscription_cancel", "Request durable cancellation with live human admin authority. HTTP 202 means accepted intent; read subscriptions for reconciled status. retry_failed explicitly reauthorizes a failed attempt.", false, true, map[string]any{"subscription_id": id("Team subscription ID"), "retry_failed": map[string]any{"type": "boolean"}}, []string{"subscription_id"}, func(ctx context.Context, c *fibe.Client, team int64, args map[string]any) (any, error) {
		subscription, err := requiredPositiveID(args, "subscription_id")
		if err != nil {
			return nil, err
		}
		retry := false
		if value, present := args["retry_failed"]; present {
			var valid bool
			retry, valid = value.(bool)
			if !valid {
				return nil, fmt.Errorf("retry_failed must be a boolean")
			}
		}
		return c.CancelTeamSubscription(ctx, team, subscription, retry)
	})
	add("fibe_team_repository_operations_list", "Read exact Team Repository operation receipts and unknown outcomes.", true, false, map[string]any{}, nil, func(ctx context.Context, c *fibe.Client, team int64, _ map[string]any) (any, error) {
		return c.TeamRepositoryOperations(ctx, team)
	})
	for _, action := range []string{"reconcile", "adopt"} {
		operation := action
		add("fibe_team_repository_operation_"+operation, "Require a live human and current Team installation authority to "+operation+" this exact remote destination. Do not overwrite/delete an unknown destination.", false, operation == "adopt", map[string]any{"request_id": map[string]any{"type": "string", "minLength": 1}, "connection_id": id("Optional current Team installation connection ID")}, []string{"request_id"}, func(ctx context.Context, c *fibe.Client, team int64, args map[string]any) (any, error) {
			var connection *int64
			if _, present := args["connection_id"]; present {
				id, err := requiredPositiveID(args, "connection_id")
				if err != nil {
					return nil, err
				}
				connection = &id
			}
			if operation == "reconcile" {
				return c.ReconcileTeamRepositoryOperation(ctx, team, argString(args, "request_id"), connection)
			}
			return c.AdoptTeamRepositoryOperation(ctx, team, argString(args, "request_id"), connection)
		})
	}
	add("fibe_team_git_installations_list", "Read current Team organization installation availability and verified authority; never export tokens.", true, false, map[string]any{}, nil, func(ctx context.Context, c *fibe.Client, team int64, _ map[string]any) (any, error) {
		return c.TeamGitInstallations(ctx, team)
	})
	add("fibe_team_git_installation_connect", "Verify a Team organization installation with live human organization-admin authority; no token export.", false, false, map[string]any{"installation_id": id("GitHub installation ID"), "verification_connection_id": id("Human verification connection ID")}, []string{"installation_id", "verification_connection_id"}, func(ctx context.Context, c *fibe.Client, team int64, args map[string]any) (any, error) {
		installation, err := requiredPositiveID(args, "installation_id")
		if err != nil {
			return nil, err
		}
		verification, err := requiredPositiveID(args, "verification_connection_id")
		if err != nil {
			return nil, err
		}
		return c.ConnectTeamGitInstallation(ctx, team, installation, verification)
	})
	add("fibe_team_git_installation_remove", "Remove a current Team installation with live human authority; reconcile held operations first.", false, true, map[string]any{"connection_id": id("Team installation connection ID")}, []string{"connection_id"}, func(ctx context.Context, c *fibe.Client, team int64, args map[string]any) (any, error) {
		connection, err := requiredPositiveID(args, "connection_id")
		if err != nil {
			return nil, err
		}
		return nil, c.RemoveTeamGitInstallation(ctx, team, connection)
	})
}
