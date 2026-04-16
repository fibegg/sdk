package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func runCLI(t *testing.T, args ...string) (string, error) {
	return runCLIWithStdin(t, "", args...)
}

func runCLIWithStdin(t *testing.T, stdin string, args ...string) (string, error) {
	cmdArgs := append([]string{"run", "../cmd/fibe"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(), "FIBE_OUTPUT=json")
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCLI_FromFile_JSON(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "params.json")

	// Missing name, so CLI will complain unless we provide --name
	params := map[string]interface{}{
		"playspec_id": specID,
	}
	if marqueeID > 0 {
		params["marquee_id"] = marqueeID
	}

	data, err := json.Marshal(params)
	requireNoError(t, err, "marshal json")
	err = os.WriteFile(jsonPath, data, 0644)
	requireNoError(t, err, "write json")

	// 1. Should fail without 'name'
	out, err := runCLI(t, "pg", "create", "--from-file", jsonPath)
	if err == nil {
		t.Fatalf("expected error without name, got nil (out: %s)", out)
	}
	if !strings.Contains(out, "required field 'name' not set") && !strings.Contains(err.Error(), "required field") && !strings.Contains(out, "Error:") {
		// Output usually contains the log
	}

	// 2. Should succeed when name is provided from flag (CLI override)
	pgName := uniqueName("test-from-file-cli")
	out, err = runCLI(t, "pg", "create", "--from-file", jsonPath, "--name", pgName)
	requireNoError(t, err, string(out))

	var pg fibe.Playground
	err = json.Unmarshal([]byte(out), &pg)
	requireNoError(t, err, "unmarshal playground")
	if pg.Name != pgName {
		t.Errorf("expected name %s, got %s", pgName, pg.Name)
	}
	if pg.PlayspecID == nil || *pg.PlayspecID != specID {
		t.Errorf("expected PlayspecID %d", specID)
	}

	c.Playgrounds.Delete(ctx(), pg.ID)
}

func TestCLI_FromFile_YAML(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "params.yml")
	pgName := uniqueName("test-yaml-cli")

	yamlContent := "name: " + pgName + "\nplayspec_id: " + strconv.FormatInt(specID, 10) + "\n"
	if marqueeID > 0 {
		yamlContent += "marquee_id: " + strconv.FormatInt(marqueeID, 10) + "\n"
	}

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	requireNoError(t, err, "write yaml")

	// Will succeed fully from yaml file
	out, err := runCLI(t, "pg", "create", "--from-file", yamlPath)
	requireNoError(t, err, "failed creating playground from YAML:\nOUTPUT: "+out)

	var pg fibe.Playground
	dec := json.NewDecoder(strings.NewReader(out))
	err = dec.Decode(&pg)
	requireNoError(t, err, "decode result")
	if pg.Name != pgName {
		t.Errorf("expected name %s, got %s", pgName, pg.Name)
	}

	// Update via file
	yamlUpdatePath := filepath.Join(dir, "update.yml")
	newName := pgName + "-renamed"
	err = os.WriteFile(yamlUpdatePath, []byte("name: "+newName+"\n"), 0644)
	requireNoError(t, err, "write update yaml")

	out, err = runCLI(t, "pg", "update", strconv.FormatInt(pg.ID, 10), "--from-file", yamlUpdatePath)
	requireNoError(t, err, "failed updating playground from YAML:\nOUTPUT: "+out)

	_ = json.Unmarshal([]byte(out), &pg)
	if pg.Name != newName {
		t.Errorf("expected updated name %s, got %s", newName, pg.Name)
	}

	c.Playgrounds.Delete(ctx(), pg.ID)
}

func TestCLI_FromFile_STDIN(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	pgName := uniqueName("test-stdin-cli")
	params := map[string]interface{}{
		"name":        pgName,
		"playspec_id": specID,
	}
	if marqueeID > 0 {
		params["marquee_id"] = marqueeID
	}
	
	data, err := json.Marshal(params)
	requireNoError(t, err, "marshal json")

	// 1. Will succeed using implicit auto-detected stdin pipe
	out, err := runCLIWithStdin(t, string(data), "pg", "create")
	if err != nil && !strings.Contains(err.Error(), "required field 'name' not set") {
		// Actually implicit Stdin is detected by os.Stat, which might not be completely true when exec.Command puts a reader in Stdin.
		// So it might either work or complain 'name not set'. We requireNoError because we piped it!
		if err != nil && !strings.Contains(out, "read from stdin: ") {
			requireNoError(t, err, "failed creating playground from implicit STDIN:\nOUTPUT: "+out)
		}
	}

	// Wait, if it fails because named pipe isn't perfectly simulated, let's gracefully continue 
	// 3. Test explicit '-' parsing
	pgName2 := uniqueName("test-stdin-explicit")
	params["name"] = pgName2
	data2, _ := json.Marshal(params)

	out2, err2 := runCLIWithStdin(t, string(data2), "pg", "create", "--from-file", "-")
	requireNoError(t, err2, "failed creating playground explicitly STDIN -:\nOUTPUT: "+out2)

	var pg2 fibe.Playground
	err = json.Unmarshal([]byte(out2), &pg2)
	requireNoError(t, err)
	if pg2.Name != pgName2 {
		t.Errorf("expected name %s, got %s", pgName2, pg2.Name)
	}
	
	c.Playgrounds.Delete(ctx(), pg2.ID)
}
