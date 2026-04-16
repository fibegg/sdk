package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

var (
	// Injected at build time via ldflags:
	//   -ldflags "-X main.version=v1.0.0 -X main.commit=abc123 -X main.date=2024-01-01"
	version = "dev"
	commit  = "none"
	date    = "unknown"

	flagAPIKey  string
	flagDomain  string
	flagDebug   bool
	flagOutput  string
	flagOnly    []string
	flagPage          int
	flagPerPage       int
	flagFromFile      string
	flagExplainErrors bool
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fibe",
		Short: "Fibe CLI — manage playgrounds, agents, and infrastructure",
		Long: `Fibe CLI is the official command-line interface for the Fibe platform API.

It provides complete access to all Fibe resources: playgrounds, tricks, agents,
playspecs, props (repositories), marquees (servers), secrets, teams, templates,
webhooks, and more.

CORE ARCHITECTURE:
  - Blueprints (Playspecs) combine services & Source Code (Props).
  - Servers (Marquees) host live environments.
  - Playgrounds are long-running environments from Playspecs on a Marquee.
  - Tricks are ad-hoc job workloads (job-mode Playspecs) that run to completion.

AUTHENTICATION:
  Set FIBE_API_KEY environment variable or use --api-key flag.
  Create API keys at https://fibe.gg or via: fibe api-keys create

EXAMPLES:
  fibe playgrounds list                      List all playgrounds
  fibe tricks list                           List all tricks (jobs)
  fibe tricks trigger --playspec-id 12       Run a trick
  fibe agents list                           List all agents
  fibe playgrounds logs 42 --service web     Stream logs for a service
  fibe me                                    Show current user info

OUTPUT:
  Default is a human-readable table. Set FIBE_OUTPUT env to change globally.
  --output table   Human-readable table (default)
  --output json    Machine-readable JSON
  --output yaml    YAML (more token-efficient for LLM contexts)
  --only id,name   Filter response to specific fields (json/yaml only)

LLM AGENT USAGE:
  export FIBE_OUTPUT=yaml
  fibe pg list --only id,name,status --page 1 --per-page 50
  fibe agents get 5 --only id,name,authenticated

ADVANCED PARAMETERS (JSON/YAML):
  Complex creation operations support loading payloads from files or standard input using --from-file (-f).
  Local config:   fibe pg create -f payload.json
  Piped input:    echo '{"name": "test"}' | fibe pg create
  Explicit "-" :  cat payload.yml | fibe pg create -f -
  Overrides:      fibe pg create -f payload.json --name "override-name"

PAGINATION:
  List commands return 25 items by default.
  Use --page and --per-page to navigate through results.
  Example: fibe playgrounds list --page 2 --per-page 10

DOCUMENTATION:
  Run any command with --help for detailed usage information.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	cmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "API key (default: $FIBE_API_KEY)")
	cmd.PersistentFlags().StringVar(&flagDomain, "domain", "", "API domain (default: $FIBE_DOMAIN, fallback: fibe.gg)")
	cmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug logging")
	cmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "Output format: table, json, yaml (default: $FIBE_OUTPUT or table)")
	cmd.PersistentFlags().StringSliceVar(&flagOnly, "only", nil, "Filter response to specific fields (e.g. --only id,name,status)")
	cmd.PersistentFlags().IntVar(&flagPage, "page", 0, "Page number for list endpoints (default: 1)")
	cmd.PersistentFlags().IntVar(&flagPerPage, "per-page", 0, "Items per page for list endpoints (default: 25)")
	cmd.PersistentFlags().StringVarP(&flagFromFile, "from-file", "f", "", "Load parameters from a JSON or YAML file (or - for STDIN)")
	cmd.PersistentFlags().BoolVar(&flagExplainErrors, "explain-errors", false, "Output errors in structured format instead of plain text string")

	cmd.AddCommand(
		meCmd(),
		playgroundsCmd(),
		tricksCmd(),
		agentsCmd(),
		playspecsCmd(),
		propsCmd(),
		marqueesCmd(),
		secretsCmd(),
		apiKeysCmd(),
		teamsCmd(),
		templatesCmd(),
		webhooksCmd(),
		mutationsCmd(),
		hunksCmd(),
		feedbacksCmd(),
		auditLogsCmd(),
		launchCmd(),
		categoriesCmd(),
		artefactsCmd(),
		muttersCmd(),
		giteaReposCmd(),
		githubReposCmd(),
		installationsCmd(),
		repoStatusCmd(),
		statusCmd(),
		limitsCmd(),
		serverInfoCmd(),
		schemaCmd(),
		waitCmd(),
		doctorCmd(),
		configCmd(),
		localPlaygroundsCmd(),
		mcpCmd(),
		versionCmd(),
		completionCmd(),
	)

	return cmd
}

func newClient() *fibe.Client {
	opts := []fibe.Option{}

	if flagAPIKey != "" {
		opts = append(opts, fibe.WithAPIKey(flagAPIKey))
	}

	if flagDomain != "" {
		opts = append(opts, fibe.WithDomain(flagDomain))
	}

	if flagDebug {
		opts = append(opts, fibe.WithDebug())
	}

	opts = append(opts, fibe.WithCircuitBreaker(fibe.DefaultBreakerConfig))
	opts = append(opts, fibe.WithRateLimitAutoWait())

	return fibe.NewClient(opts...)
}

func effectiveOutput() string {
	if flagOutput != "" {
		return flagOutput
	}
	if v := os.Getenv("FIBE_OUTPUT"); v != "" {
		return v
	}
	return "table"
}

func projectForOutput(v any, fields []string) any {
	if len(fields) == 0 {
		return v
	}

	set := make(map[string]bool)
	for _, f := range fields {
		for _, part := range strings.Split(f, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				set[trimmed] = true
			}
		}
	}
	if len(set) == 0 {
		return v
	}

	data, err := json.Marshal(v)
	if err != nil {
		return v
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return v
	}

	var dataKey string
	if _, ok := raw["data"]; ok {
		dataKey = "data"
	} else if _, ok := raw["Data"]; ok {
		dataKey = "Data"
	}

	if dataKey != "" {
		if dataSlice, ok := raw[dataKey].([]any); ok {
			for i, item := range dataSlice {
				if m, ok := item.(map[string]any); ok {
					for key := range m {
						if !set[key] {
							delete(m, key)
						}
					}
					dataSlice[i] = m
				}
			}
			raw[dataKey] = dataSlice
		}
	} else {
		for key := range raw {
			if !set[key] {
				delete(raw, key)
			}
		}
	}

	return raw
}

func output(v any) {
	if len(flagOnly) > 0 {
		v = projectForOutput(v, flagOnly)
	}
	switch effectiveOutput() {
	case "yaml":
		fmt.Print(fibe.ToYAML(v))
	default:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(v)
	}
}

func outputJSON(v any) {
	output(v)
}

func outputError(err error) {
	format := effectiveOutput()
	
	// structured format requested by explicit format or flag
	if format == "json" || format == "yaml" || flagExplainErrors {
		// Attempt to extract the API error
		var code = "UNKNOWN_ERROR"
		var details map[string]any
		var reqId string
		var statusCode = 500
		
		// If it's castable to an *APIError
		if apiErr, ok := err.(*fibe.APIError); ok {
			code = apiErr.Code
			details = apiErr.Details
			reqId = apiErr.RequestID
			statusCode = apiErr.StatusCode
		}

		errMap := map[string]any{
			"error": map[string]any{
				"message": err.Error(),
				"code":    code,
				"status":  statusCode,
			},
		}

		if details != nil {
			errMap["error"].(map[string]any)["details"] = details
		}
		if reqId != "" {
			errMap["error"].(map[string]any)["request_id"] = reqId
		}

		switch format {
		case "yaml":
			fmt.Print(fibe.ToYAML(errMap))
		default:
			enc := json.NewEncoder(os.Stderr)
			enc.SetIndent("", "  ")
			enc.Encode(errMap)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

func outputTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

func fmtTimeVal(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

func fmtBool(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func fmtBoolPtr(b *bool) string {
	if b == nil {
		return "-"
	}
	return fmtBool(*b)
}

func fmtInt64(n int64) string {
	return fmt.Sprintf("%d", n)
}

func fmtInt64Ptr(n *int64) string {
	if n == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *n)
}

func fmtStr(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

func ctx() context.Context {
	c := context.Background()
	if len(flagOnly) > 0 {
		var fields []string
		for _, f := range flagOnly {
			for _, part := range strings.Split(f, ",") {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					fields = append(fields, trimmed)
				}
			}
		}
		if len(fields) > 0 {
			c = fibe.WithFields(c, fields...)
		}
	}
	return c
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func saveDownload(body io.ReadCloser, filename string) {
	defer body.Close()
	f, err := os.Create(filename)
	must(err)
	defer f.Close()
	_, err = io.Copy(f, body)
	must(err)
	fmt.Printf("Downloaded: %s\n", filename)
}

func listParams() *fibe.ListParams {
	if flagPage == 0 && flagPerPage == 0 {
		return nil
	}
	return &fibe.ListParams{Page: flagPage, PerPage: flagPerPage}
}

