package mcpserver

import (
	"context"

	"github.com/fibegg/sdk/fibe"
)

// registerGeneratedTools wires the uniform list/get/create/update/delete
// tools for every resource in the Fibe API. Tools that require custom
// logic (wait, launch, logs --follow, pipeline, meta) are registered
// elsewhere.
//
// Tier assignment:
//   - Tools that make up common agent journeys (playgrounds, tricks, agents,
//     playspecs, status, me, schema) are tierCore.
//   - The long tail (audit logs, mutter internals, repo status, etc.) is
//     tierFull.
//
// The FIBE_MCP_TOOLS env var / --tools flag filters at registration time;
// see applyToolFilter in register.go.
func (s *Server) registerGeneratedTools() {
	// ---------- Playgrounds ----------
	registerList(s, "fibe_playgrounds_list", "List all playgrounds (excludes tricks)", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.PlaygroundListParams) (*fibe.ListResult[fibe.Playground], error) {
			f := false
			if p.JobMode == nil {
				p.JobMode = &f
			}
			return c.Playgrounds.List(ctx, p)
		})
	registerGet(s, "fibe_playgrounds_get", "Show detailed playground information", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Playground, error) {
			return c.Playgrounds.Get(ctx, id)
		})
	registerCreate(s, "fibe_playgrounds_create", "Deploy a playspec blueprint as a running playground", toolOpts{Tier: tierCore, Idempotent: true},
		func(ctx context.Context, c *fibe.Client, p *fibe.PlaygroundCreateParams) (*fibe.Playground, error) {
			return c.Playgrounds.Create(ctx, p)
		})
	registerUpdate(s, "fibe_playgrounds_update", "Update playground settings", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.PlaygroundUpdateParams) (*fibe.Playground, error) {
			return c.Playgrounds.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_playgrounds_delete", "Delete a playground (destructive, irreversible)", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Playgrounds.Delete(ctx, id)
		})
	registerIDAction(s, "fibe_playgrounds_rollout", "Recreate playground with latest configuration", toolOpts{Tier: tierCore, Destructive: true},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Playground, error) {
			return c.Playgrounds.Rollout(ctx, id)
		})
	registerIDAction(s, "fibe_playgrounds_hard_restart", "Hard restart all playground services", toolOpts{Tier: tierFull, Destructive: true},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Playground, error) {
			return c.Playgrounds.HardRestart(ctx, id)
		})
	registerIDAction(s, "fibe_playgrounds_status", "Check playground status", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.PlaygroundStatus, error) {
			return c.Playgrounds.Status(ctx, id)
		})
	registerIDAction(s, "fibe_playgrounds_compose", "Get playground docker-compose configuration", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.PlaygroundCompose, error) {
			return c.Playgrounds.Compose(ctx, id)
		})
	registerIDAction(s, "fibe_playgrounds_env", "Get playground environment metadata", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.PlaygroundEnvMetadata, error) {
			return c.Playgrounds.EnvMetadata(ctx, id)
		})
	// Debug returns map[string]any; wrap so it fits registerIDAction's generic signature.
	type pgDebug struct {
		Data map[string]any `json:"data"`
	}
	registerIDAction(s, "fibe_playgrounds_debug", "Get comprehensive debug information", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*pgDebug, error) {
			m, err := c.Playgrounds.Debug(ctx, id)
			if err != nil {
				return nil, err
			}
			return &pgDebug{Data: m}, nil
		})

	// ---------- Tricks (job-mode playgrounds) ----------
	registerList(s, "fibe_tricks_list", "List all tricks (job-mode playgrounds)", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.PlaygroundListParams) (*fibe.ListResult[fibe.Playground], error) {
			return c.Tricks.List(ctx, p)
		})
	registerGet(s, "fibe_tricks_get", "Show detailed trick information", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Playground, error) {
			return c.Tricks.Get(ctx, id)
		})
	registerCreate(s, "fibe_tricks_trigger", "Run a new trick from a job-mode playspec", toolOpts{Tier: tierCore, Idempotent: true},
		func(ctx context.Context, c *fibe.Client, p *fibe.TrickTriggerParams) (*fibe.Playground, error) {
			return c.Tricks.Trigger(ctx, p)
		})
	registerIDAction(s, "fibe_tricks_rerun", "Re-run a completed or failed trick", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Playground, error) {
			return c.Tricks.Rerun(ctx, id)
		})
	registerIDAction(s, "fibe_tricks_status", "Check trick status and job result", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.PlaygroundStatus, error) {
			return c.Tricks.Status(ctx, id)
		})
	registerDelete(s, "fibe_tricks_delete", "Delete a trick", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Tricks.Delete(ctx, id)
		})

	// ---------- Agents ----------
	registerList(s, "fibe_agents_list", "List all agents", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.AgentListParams) (*fibe.ListResult[fibe.Agent], error) {
			return c.Agents.List(ctx, p)
		})
	registerGet(s, "fibe_agents_get", "Show detailed agent information", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Agent, error) {
			return c.Agents.Get(ctx, id)
		})
	registerCreate(s, "fibe_agents_create", "Create a new agent", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.AgentCreateParams) (*fibe.Agent, error) {
			return c.Agents.Create(ctx, p)
		})
	registerUpdate(s, "fibe_agents_update", "Update agent settings", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.AgentUpdateParams) (*fibe.Agent, error) {
			return c.Agents.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_agents_delete", "Delete an agent", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Agents.Delete(ctx, id)
		})
	registerIDAction(s, "fibe_agents_duplicate", "Duplicate an agent", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Agent, error) {
			return c.Agents.Duplicate(ctx, id)
		})
	registerIDActionNoReturn(s, "fibe_agents_revoke_github", "Revoke an agent's GitHub token", toolOpts{Tier: tierFull, Destructive: true},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			_, err := c.Agents.RevokeGitHubToken(ctx, id)
			return err
		})

	// ---------- Playspecs ----------
	registerList(s, "fibe_playspecs_list", "List all playspecs", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.PlayspecListParams) (*fibe.ListResult[fibe.Playspec], error) {
			return c.Playspecs.List(ctx, p)
		})
	registerGet(s, "fibe_playspecs_get", "Show detailed playspec information", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Playspec, error) {
			return c.Playspecs.Get(ctx, id)
		})
	registerCreate(s, "fibe_playspecs_create", "Create a new playspec", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.PlayspecCreateParams) (*fibe.Playspec, error) {
			return c.Playspecs.Create(ctx, p)
		})
	registerUpdate(s, "fibe_playspecs_update", "Update a playspec", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.PlayspecUpdateParams) (*fibe.Playspec, error) {
			return c.Playspecs.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_playspecs_delete", "Delete a playspec", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Playspecs.Delete(ctx, id)
		})

	// ---------- Props (repositories) ----------
	registerList(s, "fibe_props_list", "List all props (source code repositories)", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.PropListParams) (*fibe.ListResult[fibe.Prop], error) {
			return c.Props.List(ctx, p)
		})
	registerGet(s, "fibe_props_get", "Show detailed prop information", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Prop, error) {
			return c.Props.Get(ctx, id)
		})
	registerCreate(s, "fibe_props_create", "Register a new prop (git repository)", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.PropCreateParams) (*fibe.Prop, error) {
			return c.Props.Create(ctx, p)
		})
	registerUpdate(s, "fibe_props_update", "Update a prop", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.PropUpdateParams) (*fibe.Prop, error) {
			return c.Props.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_props_delete", "Delete a prop", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Props.Delete(ctx, id)
		})
	registerIDActionNoReturn(s, "fibe_props_sync", "Sync a prop with its git remote", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Props.Sync(ctx, id)
		})

	// ---------- Marquees (servers) ----------
	registerList(s, "fibe_marquees_list", "List all marquees (servers)", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.MarqueeListParams) (*fibe.ListResult[fibe.Marquee], error) {
			return c.Marquees.List(ctx, p)
		})
	registerGet(s, "fibe_marquees_get", "Show detailed marquee information", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Marquee, error) {
			return c.Marquees.Get(ctx, id)
		})
	registerCreate(s, "fibe_marquees_create", "Register a new marquee", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.MarqueeCreateParams) (*fibe.Marquee, error) {
			return c.Marquees.Create(ctx, p)
		})
	registerUpdate(s, "fibe_marquees_update", "Update a marquee", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.MarqueeUpdateParams) (*fibe.Marquee, error) {
			return c.Marquees.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_marquees_delete", "Delete a marquee", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Marquees.Delete(ctx, id)
		})
	registerIDAction(s, "fibe_marquees_generate_ssh_key", "Generate SSH key for a marquee", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.SSHKeyResult, error) {
			return c.Marquees.GenerateSSHKey(ctx, id)
		})
	registerIDAction(s, "fibe_marquees_test_connection", "Test connectivity to a marquee", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.ConnectionTestResult, error) {
			return c.Marquees.TestConnection(ctx, id)
		})

	// ---------- Secrets ----------
	registerList(s, "fibe_secrets_list", "List all secrets", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.SecretListParams) (*fibe.ListResult[fibe.Secret], error) {
			return c.Secrets.List(ctx, p)
		})
	registerGet(s, "fibe_secrets_get", "Show secret metadata (value is redacted)", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Secret, error) {
			return c.Secrets.Get(ctx, id)
		})
	registerCreate(s, "fibe_secrets_create", "Create a secret", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.SecretCreateParams) (*fibe.Secret, error) {
			return c.Secrets.Create(ctx, p)
		})
	registerUpdate(s, "fibe_secrets_update", "Update a secret", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.SecretUpdateParams) (*fibe.Secret, error) {
			return c.Secrets.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_secrets_delete", "Delete a secret", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Secrets.Delete(ctx, id)
		})

	// ---------- API Keys ----------
	registerList(s, "fibe_api_keys_list", "List API keys", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.APIKeyListParams) (*fibe.ListResult[fibe.APIKey], error) {
			return c.APIKeys.List(ctx, p)
		})
	registerCreate(s, "fibe_api_keys_create", "Create a new API key", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.APIKeyCreateParams) (*fibe.APIKey, error) {
			return c.APIKeys.Create(ctx, p)
		})
	registerDelete(s, "fibe_api_keys_delete", "Revoke an API key", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.APIKeys.Delete(ctx, id)
		})

	// ---------- Teams ----------
	registerList(s, "fibe_teams_list", "List teams", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.TeamListParams) (*fibe.ListResult[fibe.Team], error) {
			return c.Teams.List(ctx, p)
		})
	registerGet(s, "fibe_teams_get", "Show team details", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Team, error) {
			return c.Teams.Get(ctx, id)
		})
	registerCreate(s, "fibe_teams_create", "Create a team", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.TeamCreateParams) (*fibe.Team, error) {
			return c.Teams.Create(ctx, p)
		})
	registerUpdate(s, "fibe_teams_update", "Update a team", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.TeamUpdateParams) (*fibe.Team, error) {
			return c.Teams.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_teams_delete", "Delete a team", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Teams.Delete(ctx, id)
		})
	registerIDActionNoReturn(s, "fibe_teams_leave", "Leave a team (accepts 'id' or 'team_id' for consistency with fibe_teams_members_* tools)",
		toolOpts{Tier: tierFull, Destructive: true, Aliases: map[string][]string{
			"id": {"team_id"},
		}},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.Teams.Leave(ctx, id)
		})

	// ---------- Webhook Endpoints ----------
	registerList(s, "fibe_webhooks_list", "List webhook endpoints", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.WebhookEndpointListParams) (*fibe.ListResult[fibe.WebhookEndpoint], error) {
			return c.WebhookEndpoints.List(ctx, p)
		})
	registerGet(s, "fibe_webhooks_get", "Show webhook endpoint details", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.WebhookEndpoint, error) {
			return c.WebhookEndpoints.Get(ctx, id)
		})
	registerCreate(s, "fibe_webhooks_create",
		"Create a webhook endpoint. events[] must contain exact event identifiers "+
			"(e.g. agent.created, playground.running, trick.completed) — call "+
			"fibe_webhooks_event_types first if you're not sure which strings are valid.",
		toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.WebhookEndpointCreateParams) (*fibe.WebhookEndpoint, error) {
			return c.WebhookEndpoints.Create(ctx, p)
		})
	registerUpdate(s, "fibe_webhooks_update", "Update a webhook endpoint", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.WebhookEndpointUpdateParams) (*fibe.WebhookEndpoint, error) {
			return c.WebhookEndpoints.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_webhooks_delete", "Delete a webhook endpoint", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.WebhookEndpoints.Delete(ctx, id)
		})
	registerIDActionNoReturn(s, "fibe_webhooks_test", "Send a test event to a webhook endpoint", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.WebhookEndpoints.Test(ctx, id)
		})

	// ---------- Import Templates ----------
	registerList(s, "fibe_templates_list", "List import templates", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.ImportTemplateListParams) (*fibe.ListResult[fibe.ImportTemplate], error) {
			return c.ImportTemplates.List(ctx, p)
		})
	registerGet(s, "fibe_templates_get", "Show import template details", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.ImportTemplate, error) {
			return c.ImportTemplates.Get(ctx, id)
		})
	registerCreate(s, "fibe_templates_create", "Create an import template", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.ImportTemplateCreateParams) (*fibe.ImportTemplate, error) {
			return c.ImportTemplates.Create(ctx, p)
		})
	registerUpdate(s, "fibe_templates_update", "Update an import template", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64, p *fibe.ImportTemplateUpdateParams) (*fibe.ImportTemplate, error) {
			return c.ImportTemplates.Update(ctx, id, p)
		})
	registerDelete(s, "fibe_templates_delete", "Delete an import template", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) error {
			return c.ImportTemplates.Delete(ctx, id)
		})
	// fibe_templates_launch is handled in tools_parity.go as
	// fibe_templates_launch (with marquee_id support), replacing the simple
	// registerIDAction that could not pass params.
	registerIDAction(s, "fibe_templates_fork", "Fork an import template", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.ImportTemplate, error) {
			return c.ImportTemplates.Fork(ctx, id)
		})

	// ---------- Mutations (nested under props) ----------
	registerListNested(s, "fibe_mutations_list", "List mutations for a prop", "prop_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, propID int64, p *fibe.MutationListParams) (*fibe.ListResult[fibe.Mutation], error) {
			return c.Mutations.List(ctx, propID, p)
		})
	registerCreateNested(s, "fibe_mutations_create",
		"Create a mutation for a prop. Required: branch, found_commit_sha (CLI flag --sha is accepted as an alias).",
		"prop_id",
		toolOpts{Tier: tierFull, Aliases: map[string][]string{
			"found_commit_sha": {"sha", "commit_sha", "commit"},
		}},
		func(ctx context.Context, c *fibe.Client, propID int64, p *fibe.MutationCreateParams) (*fibe.Mutation, error) {
			return c.Mutations.Create(ctx, propID, p)
		})
	registerUpdateNested(s, "fibe_mutations_update", "Update a mutation", "prop_id",
		toolOpts{Tier: tierFull, Aliases: map[string][]string{
			"found_commit_sha": {"sha", "commit_sha", "commit"},
		}},
		func(ctx context.Context, c *fibe.Client, propID, id int64, p *fibe.MutationUpdateParams) (*fibe.Mutation, error) {
			return c.Mutations.Update(ctx, propID, id, p)
		})

	// ---------- Hunks (nested under props) ----------
	registerListNested(s, "fibe_hunks_list", "List hunks for a prop", "prop_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, propID int64, p *fibe.HunkListParams) (*fibe.ListResult[fibe.Hunk], error) {
			return c.Hunks.List(ctx, propID, p)
		})
	registerGetNested(s, "fibe_hunks_get", "Show a hunk", "prop_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, propID, id int64) (*fibe.Hunk, error) {
			return c.Hunks.Get(ctx, propID, id)
		})
	registerUpdateNested(s, "fibe_hunks_update", "Update a hunk", "prop_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, propID, id int64, p *fibe.HunkUpdateParams) (*fibe.Hunk, error) {
			return c.Hunks.Update(ctx, propID, id, p)
		})

	// ---------- Feedbacks (nested under agents) ----------
	registerListNested(s, "fibe_feedbacks_list", "List feedbacks for an agent", "agent_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, agentID int64, p *fibe.FeedbackListParams) (*fibe.ListResult[fibe.Feedback], error) {
			return c.Feedbacks.List(ctx, agentID, p)
		})
	registerGetNested(s, "fibe_feedbacks_get", "Show a feedback", "agent_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, agentID, id int64) (*fibe.Feedback, error) {
			return c.Feedbacks.Get(ctx, agentID, id)
		})
	registerCreateNested(s, "fibe_feedbacks_create",
		"Create feedback for an agent. Required: source_type (e.g. \"Artefact\"), source_id (int64), selected_text, selection_start, selection_end. "+
			"source_type is the polymorphic class name from the Rails side — known values include \"Artefact\". Comment/body goes in 'comment'.",
		"agent_id",
		toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, agentID int64, p *fibe.FeedbackCreateParams) (*fibe.Feedback, error) {
			return c.Feedbacks.Create(ctx, agentID, p)
		})
	registerUpdateNested(s, "fibe_feedbacks_update", "Update feedback", "agent_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, agentID, id int64, p *fibe.FeedbackUpdateParams) (*fibe.Feedback, error) {
			return c.Feedbacks.Update(ctx, agentID, id, p)
		})
	registerDeleteNested(s, "fibe_feedbacks_delete", "Delete feedback", "agent_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, agentID, id int64) error {
			return c.Feedbacks.Delete(ctx, agentID, id)
		})

	// ---------- Artefacts (nested under agents; no create via simple schema due to io.Reader) ----------
	registerListNested(s, "fibe_artefacts_list", "List artefacts for an agent", "agent_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, agentID int64, p *fibe.ArtefactListParams) (*fibe.ListResult[fibe.Artefact], error) {
			return c.Artefacts.List(ctx, agentID, p)
		})
	registerGetNested(s, "fibe_artefacts_get", "Show an artefact", "agent_id", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, agentID, id int64) (*fibe.Artefact, error) {
			return c.Artefacts.Get(ctx, agentID, id)
		})

	// ---------- Installations ----------
	// fibe_installations_list and fibe_installations_token are registered in
	// tools_custom.go because Token needs a string "repo" param and the
	// List method takes no params at all.

	// ---------- Audit Logs (read-only) ----------
	registerList(s, "fibe_audit_logs_list", "List audit logs", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.AuditLogListParams) (*fibe.ListResult[fibe.AuditLog], error) {
			return c.AuditLogs.List(ctx, p)
		})

	// ---------- Template Categories ----------
	registerList(s, "fibe_categories_list", "List template categories", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.ListParams) (*fibe.ListResult[fibe.TemplateCategory], error) {
			return c.TemplateCategories.List(ctx, p)
		})
}
