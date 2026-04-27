package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func webhooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhooks",
		Aliases: []string{"wh"},
		Short:   "Manage webhook endpoints",
		Long: `Manage Fibe webhook endpoints — HTTP callbacks for platform events.

Webhooks deliver real-time event notifications to your URL.
Each endpoint can subscribe to specific event types and optionally
filter by resource IDs.

SUBCOMMANDS:
  list              List all webhook endpoints
  get <id>          Show endpoint details
  create            Create a new endpoint
  update <id>       Update endpoint settings
  delete <id>       Delete an endpoint
  test <id>         Send a test event
  deliveries <id>   List delivery history
  event-types       List available event types`,
	}
	cmd.AddCommand(whListCmd(), whGetCmd(), whCreateCmd(), whUpdateCmd(), whDeleteCmd(), whTestCmd(), whDeliveriesCmd(), whEventTypesCmd())
	return cmd
}

func whListCmd() *cobra.Command {
	var query, url, sort, enabled string
	cmd := &cobra.Command{
		Use: "list", Short: "List all webhook endpoints",
		Long: `List all webhook endpoints for the authenticated user.

FILTERS:
  -q, --query           Search across url, description (substring match)
  --enabled             Filter by enabled state. Values: true, false
  --url                 Filter by URL (substring match)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, URL, ENABLED, FAILURES, EVENTS
  Use --output json for full details.

EXAMPLES:
  fibe webhooks list
  fibe wh list --enabled true
  fibe wh list -q "example.com" -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.WebhookEndpointListParams{}
			if query != "" {
				params.Q = query
			}
			if enabled == "true" {
				t := true
				params.Enabled = &t
			} else if enabled == "false" {
				f := false
				params.Enabled = &f
			}
			if url != "" {
				params.URL = url
			}
			if sort != "" {
				params.Sort = sort
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			eps, err := c.WebhookEndpoints.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(eps)
				return nil
			}
			headers := []string{"ID", "URL", "ENABLED", "FAILURES", "EVENTS"}
			rows := make([][]string, len(eps.Data))
			for i, e := range eps.Data {
				evts := fmt.Sprintf("%d events", len(e.Events))
				rows[i] = []string{fmtInt64Ptr(e.ID), e.URL, fmtBoolPtr(e.Enabled), fmtInt64Ptr(e.FailureCount), evts}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across url, description")
	cmd.Flags().StringVar(&enabled, "enabled", "", "Filter by enabled state (true/false)")
	cmd.Flags().StringVar(&url, "url", "", "Filter by URL (substring)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func whGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <id>", Short: "Show endpoint details", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			ep, err := c.WebhookEndpoints.Get(ctx(), id)
			if err != nil {
				return err
			}
			outputJSON(ep)
			return nil
		},
	}
}

func whCreateCmd() *cobra.Command {
	var urlStr, secret, desc string
	var events, eventFilters, toolFilters []string
	cmd := &cobra.Command{
		Use: "create", Short: "Create a new webhook endpoint",
		Long: `Create a new webhook endpoint.

WEBHOOK RULES:
  - Fibe actively prevents SSRF; internal network IPs are strictly prohibited.
  - 10 consecutive failed deliveries to your host will auto-disable the webhook.

REQUIRED FLAGS:
  --url             Delivery URL (HTTPS)
  --event           Event types to subscribe (repeatable)

OPTIONAL FLAGS:
  --secret          Signing secret (auto-generated when omitted)
  --description     Endpoint description
  --event-filter    Restrict an event to specific resource IDs (repeatable)
                    Format: event=id1,id2,...
                    e.g. --event-filter playground.created=12,15
  --tool-filter     Restrict an event to specific MCP tool names (repeatable)
                    Format: event=tool1,tool2,...
                    e.g. --tool-filter mcp.tool.executed=deploy,status

For complex event_filters/tool_filters, use --from-file with JSON.

EXAMPLES:
  fibe webhooks create --url https://example.com/hook \
    --event playground.created --event playground.status.changed
  fibe webhooks create --url https://example.com/hook --secret mysecret \
    --event playground.created --event-filter playground.created=12,15`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.WebhookEndpointCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("url") {
				params.URL = urlStr
			}
			if cmd.Flags().Changed("secret") {
				params.Secret = secret
			}
			if cmd.Flags().Changed("event") {
				params.Events = events
			}
			if cmd.Flags().Changed("description") {
				params.Description = &desc
			}
			if len(eventFilters) > 0 {
				if params.EventFilters == nil {
					params.EventFilters = map[string]any{}
				}
				for _, ef := range eventFilters {
					eq := strings.Index(ef, "=")
					if eq < 1 || eq == len(ef)-1 {
						return fmt.Errorf("invalid --event-filter %q: expected event=id1,id2,...", ef)
					}
					name := ef[:eq]
					ids := []int64{}
					for _, idStr := range strings.Split(ef[eq+1:], ",") {
						id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
						if err != nil {
							return fmt.Errorf("invalid id %q in --event-filter %q: %w", idStr, ef, err)
						}
						ids = append(ids, id)
					}
					params.EventFilters[name] = ids
				}
			}
			if len(toolFilters) > 0 {
				if params.ToolFilters == nil {
					params.ToolFilters = map[string][]string{}
				}
				for _, tf := range toolFilters {
					eq := strings.Index(tf, "=")
					if eq < 1 || eq == len(tf)-1 {
						return fmt.Errorf("invalid --tool-filter %q: expected event=tool1,tool2,...", tf)
					}
					name := tf[:eq]
					tools := []string{}
					for _, tool := range strings.Split(tf[eq+1:], ",") {
						trimmed := strings.TrimSpace(tool)
						if trimmed != "" {
							tools = append(tools, trimmed)
						}
					}
					params.ToolFilters[name] = tools
				}
			}

			if params.URL == "" {
				return fmt.Errorf("required field 'url' not set")
			}
			if len(params.Events) == 0 {
				return fmt.Errorf("required field 'event' not set")
			}

			ep, err := c.WebhookEndpoints.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(ep)
				return nil
			}
			fmt.Printf("Created webhook endpoint %s\n", fmtInt64Ptr(ep.ID))
			return nil
		},
	}
	cmd.Flags().StringVar(&urlStr, "url", "", "Delivery URL (required)")
	cmd.Flags().StringVar(&secret, "secret", "", "Signing secret (optional; auto-generated when omitted)")
	cmd.Flags().StringSliceVar(&events, "event", nil, "Event types (required, repeatable)")
	cmd.Flags().StringSliceVar(&eventFilters, "event-filter", nil, "Event filter (repeatable, format: event=id1,id2)")
	cmd.Flags().StringSliceVar(&toolFilters, "tool-filter", nil, "Tool filter (repeatable, format: event=tool1,tool2)")
	cmd.Flags().StringVar(&desc, "description", "", "Description")
	return cmd
}

func whUpdateCmd() *cobra.Command {
	var urlStr, secret, description string
	var enabled bool
	var events, eventFilters, toolFilters []string
	cmd := &cobra.Command{
		Use: "update <id>", Short: "Update endpoint settings", Args: cobra.ExactArgs(1),
		Long: `Update an existing webhook endpoint.

OPTIONAL FLAGS:
  --url             New endpoint URL
  --secret          New HMAC signing secret
  --enabled         Enable/disable the endpoint
  --description     New description
  --event           Subscribed events (repeatable, replaces existing list)
  --event-filter    Event filter (repeatable, format: event=id1,id2,...)
                    e.g. --event-filter playground.created=12,15
  --tool-filter     MCP tool filter (repeatable, format: event=tool1,tool2,...)
                    e.g. --tool-filter mcp.tool.executed=deploy,status

For complex event_filters/tool_filters, use --from-file with JSON.

EXAMPLES:
  fibe webhooks update 5 --enabled=false
  fibe wh update 5 --url https://hooks.example.com/new --secret newSecret
  fibe wh update 5 --event playground.created --event marquee.deleted
  fibe wh update 5 --event-filter playground.created=12,15`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.WebhookEndpointUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("url") {
				params.URL = &urlStr
			}
			if cmd.Flags().Changed("secret") {
				params.Secret = &secret
			}
			if cmd.Flags().Changed("description") {
				params.Description = &description
			}
			if cmd.Flags().Changed("enabled") {
				params.Enabled = &enabled
			}
			if len(events) > 0 {
				params.Events = events
			}
			if len(eventFilters) > 0 {
				if params.EventFilters == nil {
					params.EventFilters = map[string]any{}
				}
				for _, ef := range eventFilters {
					eq := strings.Index(ef, "=")
					if eq < 1 || eq == len(ef)-1 {
						return fmt.Errorf("invalid --event-filter %q: expected event=id1,id2,...", ef)
					}
					name := ef[:eq]
					ids := []int64{}
					for _, idStr := range strings.Split(ef[eq+1:], ",") {
						id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
						if err != nil {
							return fmt.Errorf("invalid id %q in --event-filter %q: %w", idStr, ef, err)
						}
						ids = append(ids, id)
					}
					params.EventFilters[name] = ids
				}
			}
			if len(toolFilters) > 0 {
				if params.ToolFilters == nil {
					params.ToolFilters = map[string][]string{}
				}
				for _, tf := range toolFilters {
					eq := strings.Index(tf, "=")
					if eq < 1 || eq == len(tf)-1 {
						return fmt.Errorf("invalid --tool-filter %q: expected event=tool1,tool2,...", tf)
					}
					name := tf[:eq]
					tools := []string{}
					for _, tool := range strings.Split(tf[eq+1:], ",") {
						trimmed := strings.TrimSpace(tool)
						if trimmed != "" {
							tools = append(tools, trimmed)
						}
					}
					params.ToolFilters[name] = tools
				}
			}
			ep, err := c.WebhookEndpoints.Update(ctx(), id, params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(ep)
				return nil
			}
			fmt.Printf("Updated webhook endpoint %d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&urlStr, "url", "", "New URL")
	cmd.Flags().StringVar(&secret, "secret", "", "New HMAC signing secret")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable/disable")
	cmd.Flags().StringSliceVar(&events, "event", nil, "Subscribed event (repeatable)")
	cmd.Flags().StringSliceVar(&eventFilters, "event-filter", nil, "Event filter (repeatable, format: event=id1,id2)")
	cmd.Flags().StringSliceVar(&toolFilters, "tool-filter", nil, "Tool filter (repeatable, format: event=tool1,tool2)")
	return cmd
}

func whDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a webhook endpoint", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.WebhookEndpoints.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Webhook endpoint %d deleted\n", id)
			return nil
		},
	}
}

func whTestCmd() *cobra.Command {
	return &cobra.Command{
		Use: "test <id>", Short: "Send test event to endpoint", Args: cobra.ExactArgs(1),
		Long: "Queue a test webhook delivery to verify the endpoint is working.\n\nEXAMPLES:\n  fibe webhooks test 3",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.WebhookEndpoints.Test(ctx(), id); err != nil {
				return err
			}
			fmt.Println("Test event queued")
			return nil
		},
	}
}

func whDeliveriesCmd() *cobra.Command {
	return &cobra.Command{
		Use: "deliveries <id>", Short: "List delivery history", Args: cobra.ExactArgs(1),
		Long: "List webhook delivery attempts for an endpoint.\n\nEXAMPLES:\n  fibe webhooks deliveries 3",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			deliveries, err := c.WebhookEndpoints.ListDeliveries(ctx(), id, nil)
			if err != nil {
				return err
			}
			outputJSON(deliveries)
			return nil
		},
	}
}

func whEventTypesCmd() *cobra.Command {
	return &cobra.Command{
		Use: "event-types", Short: "List available event types",
		Long: "List all event types that can be subscribed to.\n\nEXAMPLES:\n  fibe webhooks event-types",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			types, err := c.WebhookEndpoints.EventTypes(ctx())
			if err != nil {
				return err
			}
			for _, t := range types {
				fmt.Println(t)
			}
			return nil
		},
	}
}
