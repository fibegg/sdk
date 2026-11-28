package mcpserver

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/fibegg/sdk/contracts"
)

type restOperation struct {
	Name     string `json:"name"`
	Method   string `json:"http_method"`
	Path     string `json:"path_template"`
	Delegate string `json:"delegates_to"`
}

var restRoutes = sync.OnceValue(func() map[string]restOperation {
	var contract struct {
		Operations []restOperation `json:"operations"`
	}
	if err := json.Unmarshal([]byte(contracts.REST()), &contract); err != nil {
		panic(err)
	}
	index := make(map[string]restOperation, len(contract.Operations))
	for _, op := range contract.Operations {
		index[op.Name] = op
	}
	return index
})

// Each clause describes one operation. A space separates alternatives and a
// plus joins the routes an operation needs before its first request. "-" is local.
type operationRoutes string

func operationRoute(id string) apiRoute {
	if method, path, ok := strings.Cut(id, ":"); ok {
		return routeKey(method, path)
	}
	for range 10 {
		if route := restRouteOverrides[id]; route != "" {
			return operationRoute(route)
		}
		op := restRoutes()[id]
		if op.Method != "" {
			path, _, _ := strings.Cut(strings.ReplaceAll(op.Path, "{?query}", ""), "?")
			if strings.HasPrefix(path, "/") {
				return routeKey(op.Method, path)
			}
			break
		}
		if op.Delegate == "" {
			break
		}
		id = normalizeRESTDelegate(op.Delegate)
	}
	return apiRoute{}
}

func (m *routeManifest) offers(routes operationRoutes) bool {
	if m == nil || routes == "-" || routes == "" {
		return true
	}
	for _, clause := range strings.Fields(string(routes)) {
		present := true
		for _, id := range strings.Split(clause, "+") {
			if route := operationRoute(id); route.path != "" && !m.paths[route] {
				present = false
				break
			}
		}
		if present {
			return true
		}
	}
	return false
}

type unavailableOperation string

func (e unavailableOperation) Error() string { return string(e) }

func (m *routeManifest) require(operation string, routes operationRoutes) error {
	if m.offers(routes) {
		return nil
	}
	return unavailable(operation)
}

func unavailable(operation string) error {
	return unavailableOperation(operation + " is not available: this server runs the Community edition.")
}

const (
	templatePatchPreview   = "ImportTemplateService.PatchPreview"
	templatePatchApply     = "ImportTemplateService.PatchCreate"
	templateSwitchPreview  = "SpecService.PreviewTemplateVersionSwitchByIdentifier"
	templateSwitchApply    = "SpecService.SwitchTemplateVersionByIdentifier"
	templateChangeRoutes   = templatePatchPreview + " " + templatePatchApply + " " + templateSwitchPreview + " " + templateSwitchApply
	playgroundSwitchRoutes = "PlaygroundService.GetByIdentifier+" + templateSwitchPreview + " PlaygroundService.GetByIdentifier+" + templateSwitchApply
	templateUpdateRoutes   = "ImportTemplateService.UpdateByIdentifier ImportTemplateService.UploadImageByIdentifier"
	launchRoutes           = "ImportTemplateService.LaunchWithParamsByIdentifier GreenfieldService.Create PlaygroundService.Create LaunchService.Create"
	followPlaygroundLogs   = "PlaygroundService.GetByIdentifier+GET:/cable"
	followTaskLogs         = "TaskService.GetByIdentifier+GET:/cable"
	followLogsRoutes       = followPlaygroundLogs + " " + followTaskLogs
	artefactUploadRoutes   = "ArtefactService.CreateOwned ArtefactService.CreateByAgentIdentifier"

	envPackPlayground    = "/api/playgrounds/{id}/env_pack_attachments"
	envPackSpec          = "/api/specs/{id}/env_pack_attachments"
	envPackGetRoutes     = "GET:" + envPackPlayground + " GET:" + envPackSpec
	envPackReplaceRoutes = "PATCH:" + envPackPlayground + " PATCH:" + envPackSpec
	envPackRenewRoutes   = "POST:" + envPackPlayground + "/renew POST:" + envPackSpec + "/renew"
	envPackModifyRoutes  = "GET:" + envPackPlayground + "+PATCH:" + envPackPlayground + " GET:" + envPackSpec + "+PATCH:" + envPackSpec
)

// These SDK methods have dynamic, paginated or composed paths that rest.json
// does not represent as a usable template. Keep these entries to the paths
// used by their client implementations.
// #nosec G101 -- the table contains SDK method names and public route templates.
var restRouteOverrides = map[string]string{
	"APIKeyService.List":                                      "GET:/api/api_keys",
	"AgentService.List":                                       "GET:/api/agents",
	"AgentService.ListPokesByIdentifier":                      "GET:/api/agents/{id}/pokes",
	"AgentService.GetPokeByIdentifier":                        "GET:/api/agents/{id}/pokes/{pokeID}",
	"AgentService.UpdatePokeByIdentifier":                     "PATCH:/api/agents/{id}/pokes/{pokeID}",
	"AgentService.DeletePokeByIdentifier":                     "DELETE:/api/agents/{id}/pokes/{pokeID}",
	"ArtefactService.ListAll":                                 "GET:/api/artefacts",
	"AuditLogService.List":                                    "GET:/api/audit_logs",
	"EnvPackService.List":                                     "GET:/api/env_packs",
	"HostService.List":                                        "GET:/api/hosts",
	"ImportTemplateService.List":                              "GET:/api/import_templates",
	"ImportTemplateService.SearchWithParams":                  "GET:/api/import_templates",
	"ImportTemplateService.ListVersionsByIdentifier":          "GET:/api/import_templates/{id}/versions",
	"ImportTemplateService.TogglePublicByIdentifier":          "PATCH:/api/import_templates/{id}/versions/{versionID}/publication",
	"ImportTemplateService.UpgradeLinkedSpecsByIdentifier":    "POST:/api/import_templates/{id}/versions/{versionID}/upgrades",
	"ImportTemplateVersionService.DeleteByTemplateIdentifier": "DELETE:/api/import_templates/{id}/versions/{versionID}",
	"JobEnvService.List":                                      "GET:/api/job_env",
	"JobEnvService.Get":                                       "GET:/api/job_env/{id}",
	"MemoryService.List":                                      "GET:/api/memories",
	"PlaygroundService.List":                                  "GET:/api/playgrounds",
	"RepositoryService.List":                                  "GET:/api/repositories",
	"SecretService.List":                                      "GET:/api/secrets",
	"SecretService.GetByIdentifier":                           "GET:/api/secrets/{id}",
	"SpecService.List":                                        "GET:/api/specs",
	"TaskService.Trigger":                                     "POST:/api/playgrounds",
	"TaskService.RerunWithParamsByIdentifier":                 "POST:/api/playgrounds/{id}/rerun",
	"TaskService.List":                                        "GET:/api/playgrounds",
	"TemplateCategoryService.List":                            "GET:/api/template_categories",
	"WebhookEndpointService.List":                             "GET:/api/webhook_endpoints",
	"WebhookEndpointService.ListDeliveries":                   "GET:/api/webhook_endpoints/{id}/deliveries",
}

func normalizeRESTDelegate(delegate string) string {
	service, method, ok := strings.Cut(delegate, ".")
	if ok && service == "Playgrounds" {
		return "PlaygroundService." + method
	}
	if ok && service == "Specs" {
		return "SpecService." + method
	}
	return delegate
}
