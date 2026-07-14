package fibe

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func TestCredentialStoreIsAtomicPrivateAndReturnsCopies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "credentials.json")
	store := NewCredentialStore(path)
	entry := &CredentialEntry{APIKey: "secret", Domain: "fibe.gg"}
	if err := store.Set(entry); err != nil {
		t.Fatal(err)
	}
	entry.APIKey = "mutated"
	got, err := store.Get("fibe.gg")
	if err != nil || got.APIKey != "secret" {
		t.Fatalf("stored entry=%#v err=%v", got, err)
	}
	got.APIKey = "caller mutation"
	again, err := store.Get("fibe.gg")
	if err != nil || again.APIKey != "secret" {
		t.Fatalf("store leaked mutable entry=%#v err=%v", again, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("credential mode=%v err=%v", info.Mode().Perm(), err)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil || dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("credential dir mode=%v err=%v", dirInfo.Mode().Perm(), err)
	}
}

func TestCredentialStoreConcurrentUpdatesDoNotGetLost(t *testing.T) {
	store := NewCredentialStore(filepath.Join(t.TempDir(), "credentials.json"))
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			entry := &CredentialEntry{APIKey: fmt.Sprintf("key-%d", i), Domain: fmt.Sprintf("domain-%d", i)}
			if err := store.SetProfile(fmt.Sprintf("profile-%d", i), entry); err != nil {
				t.Errorf("set profile: %v", err)
			}
		}()
	}
	wg.Wait()
	profiles, err := store.ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		if profiles[fmt.Sprintf("profile-%d", i)] == nil {
			t.Fatalf("profile-%d was lost", i)
		}
	}
}

func TestAuthProfileStoreConcurrentUpdatesAndSymlinkRejection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewAuthProfileStore(path)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := store.SetProfile(fmt.Sprintf("profile-%d", i), fmt.Sprintf("domain-%d", i)); err != nil {
				t.Errorf("set profile: %v", err)
			}
		}()
	}
	wg.Wait()
	cfg, err := store.Load()
	if err != nil || len(cfg.Profiles) != 50 {
		t.Fatalf("profiles=%d err=%v", len(cfg.Profiles), err)
	}

	target := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(target, []byte(`{"profiles":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "config.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	linked := NewAuthProfileStore(link)
	if _, err := linked.Load(); err == nil {
		t.Fatal("symlink store was read")
	}
	if err := linked.Save(&AuthProfileConfig{}); err == nil {
		t.Fatal("symlink store was overwritten")
	}
}

func TestCredentialStoreConcurrentProcessesDoNotLoseUpdates(t *testing.T) {
	if path := os.Getenv("FIBE_STORE_HELPER_PATH"); path != "" {
		index, err := strconv.Atoi(os.Getenv("FIBE_STORE_HELPER_INDEX"))
		if err != nil {
			t.Fatal(err)
		}
		entry := &CredentialEntry{APIKey: fmt.Sprintf("key-%d", index), Domain: fmt.Sprintf("domain-%d", index)}
		if err := NewCredentialStore(path).SetProfile(fmt.Sprintf("profile-%d", index), entry); err != nil {
			t.Fatal(err)
		}
		return
	}

	path := filepath.Join(t.TempDir(), "credentials.json")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	const processCount = 12
	type runningCommand struct {
		command *exec.Cmd
		output  *bytes.Buffer
	}
	commands := make([]runningCommand, 0, processCount)
	for i := 0; i < processCount; i++ {
		command := exec.Command(executable, "-test.run=^TestCredentialStoreConcurrentProcessesDoNotLoseUpdates$")
		command.Env = append(os.Environ(),
			"FIBE_STORE_HELPER_PATH="+path,
			"FIBE_STORE_HELPER_INDEX="+strconv.Itoa(i),
		)
		output := &bytes.Buffer{}
		command.Stdout = output
		command.Stderr = output
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, runningCommand{command: command, output: output})
	}
	for _, running := range commands {
		if err := running.command.Wait(); err != nil {
			t.Fatalf("credential writer failed: %v\n%s", err, running.output.Bytes())
		}
	}
	profiles, err := NewCredentialStore(path).ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < processCount; i++ {
		if profiles[fmt.Sprintf("profile-%d", i)] == nil {
			t.Fatalf("profile-%d was lost across concurrent processes", i)
		}
	}
}
