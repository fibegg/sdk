package mcpserver

import (
	"context"
	"os"
	"strings"
)

// RunAPIKeyEnv carries an explicit session API key from fibe_run to the CLI
// subprocess. The CLI treats it like --api-key, so the key never appears in
// argv, where other local users could read it, or in the fibe_run result.
//
// #nosec G101 -- this names an environment variable; it contains no credential value.
const RunAPIKeyEnv = "FIBE_MCP_RUN_API_KEY"

// fibeRunAuth returns the CLI flags and environment that make a fibe_run
// subprocess use the same credentials as this session's client. A selected
// profile is passed by name so the child reads its own key from the store;
// any other key (fibe_auth_set, HTTP bearer, --api-key or FIBE_API_KEY on the
// server) is passed through RunAPIKeyEnv. Flags the caller already passed in
// userArgs win.
func (s *Server) fibeRunAuth(ctx context.Context, userArgs []string) ([]string, []string) {
	st := s.sessionFor(ctx)
	st.mu.RLock()
	profile, sessionKey := st.profile, st.apiKey
	apiKey, domain := st.effectiveAPIKey, st.effectiveDomain
	st.mu.RUnlock()

	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, RunAPIKeyEnv+"=") {
			env = append(env, entry)
		}
	}

	var flags []string
	if !containsCLIFlag(userArgs, "--domain") && domain != "" {
		flags = append(flags, "--domain", domain)
	}
	if containsCLIFlag(userArgs, "--profile") || containsCLIFlag(userArgs, "--api-key") {
		return flags, env
	}
	if profile == "" && sessionKey == "" && apiKeyFromContext(ctx) == "" && s.cfg.Profile != "" &&
		(s.cfg.AuthSource == "profile "+s.cfg.Profile || s.cfg.APIKey == "") {
		profile = s.cfg.Profile
	}
	if profile != "" {
		return append(flags, "--profile", profile), env
	}
	if apiKey != "" {
		env = append(env, RunAPIKeyEnv+"="+apiKey)
	}
	return flags, env
}
