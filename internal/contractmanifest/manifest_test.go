package contractmanifest

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestCurrentPublicAPIManifestIsExact(t *testing.T) {
	root := repositoryRoot(t)
	generated, err := Generate(filepath.Join(root, "fibe"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "contracts", "go-public-api.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, want) {
		t.Fatal("public Go API manifest is stale; run go run ./internal/cmd/apimanifest")
	}
}

func TestV03PreservesExistingAPIOutsideApprovedFeatures(t *testing.T) {
	root := repositoryRoot(t)
	baseline, err := Read(filepath.Join(root, "contracts", "go-public-api-v0.2.45.json"))
	if err != nil {
		t.Fatal(err)
	}
	currentData, err := Generate(filepath.Join(root, "fibe"))
	if err != nil {
		t.Fatal(err)
	}
	currentPath := filepath.Join(t.TempDir(), "current.json")
	if err := os.WriteFile(currentPath, currentData, 0o600); err != nil {
		t.Fatal(err)
	}
	current, err := Read(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	// The v0.2.45 artifact remains immutable. Its declarations are translated
	// only for the approved v0.3 domain break. Exact additive Teams and ENV-pack
	// fields are projected out below; all existing fields and methods stay checked.
	for i := range baseline.Declarations {
		baseline.Declarations[i].ID = canonicalDomainDeclaration(baseline.Declarations[i].ID)
		baseline.Declarations[i].Signature = strings.Join(strings.Fields(canonicalDomainDeclaration(baseline.Declarations[i].Signature)), " ")
	}
	for i := range current.Declarations {
		declaration := &current.Declarations[i]
		declaration.Signature = strings.Join(strings.Fields(withoutApprovedFeatureFields(*declaration)), " ")
	}
	if problems := CompatibilityErrors(baseline, current); len(problems) > 0 {
		t.Fatalf("unexpected changes outside the approved v0.3 features:\n%s", strings.Join(problems, "\n"))
	}
}

// Match the whole field declaration, including its type and JSON contract. A
// changed existing field or an unapproved addition still fails compatibility.
func withoutApprovedFeatureFields(declaration Declaration) string {
	allowed := make(map[string]bool)
	switch declaration.ID {
	case "type.Agent", "type.Artefact", "type.Feedback", "type.Host",
		"type.ImportTemplate", "type.ImportTemplateVersion", "type.JobEnvEntry",
		"type.Memory", "type.Mutter", "type.Playground", "type.Repository",
		"type.Secret", "type.Spec", "type.WebhookEndpoint":
		allowed["OwnershipMetadata"] = true
	case "type.APIKey", "type.CredentialEntry", "type.Player":
		allowed["CredentialContext"] = true
	case "type.APIKeyCreateParams":
		allowed["PrincipalType string `json:\"principal_type,omitempty\"`"] = true
		allowed["PrincipalID *int64 `json:\"principal_id,omitempty\"`"] = true
	case "type.Client":
		allowed["EnvPacks *EnvPackService"] = true
	}
	switch declaration.ID {
	case "type.Playground", "type.Spec":
		allowed["EnvPackAttachments []EnvPackAttachment `json:\"env_pack_attachments,omitempty\"`"] = true
		allowed["EnvPacks map[string]any `json:\"env_packs,omitempty\"`"] = true
	case "type.PlaygroundCreateParams", "type.PlaygroundUpdateParams",
		"type.SpecCreateParams", "type.SpecUpdateParams", "type.TaskTriggerParams",
		"type.LaunchParams", "type.ImportTemplateLaunchParams", "type.GreenfieldCreateParams":
		allowed["EnvPackAttachments *[]EnvPackAttachmentInput `json:\"env_pack_attachments,omitempty\"`"] = true
	}
	var lines []string
	for _, line := range strings.Split(declaration.Signature, "\n") {
		if !allowed[strings.Join(strings.Fields(line), " ")] {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func canonicalDomainDeclaration(value string) string {
	replacements := strings.NewReplacer(
		"Marquees", "Hosts", "Marquee", "Host", "marquees", "hosts", "marquee", "host",
		"Playspecs", "Specs", "Playspec", "Spec", "playspecs", "specs", "playspec", "spec",
		"Tricks", "Tasks", "Trick", "Task", "tricks", "tasks", "trick", "task",
		"MARQUEE", "HOST", "PLAYSPEC", "SPEC", "PROP_", "REPOSITORY_", "TRICK", "TASK",
	)
	value = replacements.Replace(value)
	value = strings.ReplaceAll(value, "prop_", "repository_")
	value = strings.ReplaceAll(value, "props_", "repositories_")
	value = strings.ReplaceAll(value, "_props", "_repositories")
	value = strings.ReplaceAll(value, "_prop", "_repository")
	value = regexp.MustCompile(`Props([A-Z]|\b)`).ReplaceAllString(value, `Repositories${1}`)
	value = regexp.MustCompile(`Prop([A-Z]|\b)`).ReplaceAllString(value, `Repository${1}`)
	value = regexp.MustCompile(`(^|_)prop_`).ReplaceAllString(value, `${1}repository_`)
	value = regexp.MustCompile(`\bprops\b`).ReplaceAllString(value, "repositories")
	value = regexp.MustCompile(`\bprop\b`).ReplaceAllString(value, "repository")
	value = regexp.MustCompile(`\bprop([A-Z])`).ReplaceAllString(value, `repository${1}`)
	return value
}
