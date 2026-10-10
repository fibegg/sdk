package integration

// ENV packs end to end against a live Rails: the Go SDK, the built fibe CLI
// binary and the fibe MCP stdio server are driven through the same scenarios
// for pack CRUD (Personal and Company), attachment set/reorder/retarget/detach/
// renew, delegated sources, launch with attachments, safe Playground and Task
// reruns, nil versus empty values, and the four stable error codes.
//
// Company credential: RUNTIME_API_KEY, RUNTIME_OWNER_TYPE and RUNTIME_OWNER_ID
// (and RUNTIME_HOST_ID for Company Playground targets) are exported by the
// Playwright bootstrap. The app-go-tests lane bootstraps mode=sdk, which does
// not export them, so Company coverage skips with an explicit reason unless
// they are set. FIBE_REQUIRE_COMPANY_ENV_PACKS=1 turns that skip into a failure.
//
// The file carries no build tag, like every other file in this package: the
// app-go-tests lane runs ./integration/... without -tags, so a tag would make
// it silently skip there. Without FIBE_INTEGRATION/FIBE_E2E_BOOTSTRAP every test
// skips through userClient.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const missingEnvPackID int64 = 2_000_000_000

// forEachEnvPackContext runs fn as a subtest for the Personal and then the
// Company context. A missing Company credential skips with a reason.
func forEachEnvPackContext(t *testing.T, fn func(t *testing.T, a *envPackActor)) {
	t.Helper()
	t.Run("personal", func(t *testing.T) { fn(t, personalEnvPackActor(t)) })
	t.Run("company", func(t *testing.T) { fn(t, companyEnvPackActor(t)) })
}

func forEachSurface(t *testing.T, a *envPackActor, fn func(t *testing.T, s envPackSurface)) {
	t.Helper()
	for _, s := range surfacesFor(t, a) {
		s := s
		t.Run(s.Name(), func(t *testing.T) { fn(t, s) })
	}
}

func requireEnv(t *testing.T, surface string, got map[string]string, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: env %q, want %q", surface, got, want)
	}
	for key, value := range want {
		if actual, ok := got[key]; !ok || actual != value {
			t.Fatalf("%s: env[%s] = %q (present=%v), want %q; full env %q", surface, key, actual, ok, value, got)
		}
	}
}

// envPackTargetHost returns the Host for Playground targets: the lane Host for
// Personal, the Company Host exported by the Playwright bootstrap for Company.
func envPackTargetHost(t *testing.T, a *envPackActor) int64 {
	t.Helper()
	if a.label == "personal" {
		return envPackHost(t)
	}
	raw := strings.TrimSpace(os.Getenv("RUNTIME_HOST_ID"))
	if raw == "" {
		reason := "a Company Playground target needs RUNTIME_HOST_ID (the Company Host exported by the Playwright bootstrap)"
		if envBool("FIBE_REQUIRE_COMPANY_ENV_PACKS") {
			t.Fatal(reason)
		}
		t.Skip(reason)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		t.Fatalf("invalid RUNTIME_HOST_ID %q", raw)
	}
	return id
}

// ------------------------------------------------------------------ pack CRUD

func TestEnvPacks_CRUD(t *testing.T) {
	forEachEnvPackContext(t, func(t *testing.T, a *envPackActor) {
		forEachSurface(t, a, func(t *testing.T, s envPackSurface) {
			values := map[string]string{
				"EMPTY":     "",
				"QUOTED":    `say "hi" and 'bye'`,
				"MULTILINE": "line one\nline two",
				"PADDED":    "  padded  ",
			}
			name := uniqueName("envpack-crud-" + s.Name())
			pack, err := s.CreatePack(t, name, values)
			requireNoError(t, err, s.Name()+" create")
			t.Cleanup(func() { _ = a.client.EnvPacks.Delete(ctx(), pack.ID) })

			if pack.ID == 0 || pack.Name != name {
				t.Fatalf("%s: created pack %+v", s.Name(), pack)
			}
			if pack.OwnerType != a.ownerType || pack.OwnerID != a.ownerID {
				t.Fatalf("%s: pack owner %s:%d, want %s:%d", s.Name(), pack.OwnerType, pack.OwnerID, a.ownerType, a.ownerID)
			}
			requireEnv(t, s.Name()+" create", pack.Env, values)

			got, err := s.GetPack(t, pack.ID)
			requireNoError(t, err, s.Name()+" get")
			requireEnv(t, s.Name()+" get", got.Env, values)

			t.Run("list", func(t *testing.T) {
				rows, err := s.ListPacks(t)
				requireNoError(t, err, s.Name()+" list")
				found := false
				for _, row := range rows {
					if row.ID == pack.ID {
						found = true
					}
					if row.OwnerType != a.ownerType || row.OwnerID != a.ownerID {
						t.Errorf("%s: list returned a pack outside the %s context: %+v", s.Name(), a.label, row)
					}
				}
				if !found {
					t.Fatalf("%s: created pack %d missing from list", s.Name(), pack.ID)
				}
			})

			// Omitted env preserves; the name alone changes.
			renamed := name + "-renamed"
			updated, err := s.UpdatePack(t, pack.ID, &fibe.EnvPackUpdateParams{Name: &renamed})
			requireNoError(t, err, s.Name()+" rename")
			if updated.Name != renamed {
				t.Fatalf("%s: name %q, want %q", s.Name(), updated.Name, renamed)
			}
			requireEnv(t, s.Name()+" rename preserves env", updated.Env, values)

			// Replacing env with the expected version keeps empty strings and bumps the version.
			next := map[string]string{"EMPTY": "", "ADDED": "new\nvalue"}
			replaced, err := s.UpdatePack(t, pack.ID, &fibe.EnvPackUpdateParams{Env: &next, Version: &updated.Version})
			requireNoError(t, err, s.Name()+" replace env")
			requireEnv(t, s.Name()+" replace env", replaced.Env, next)
			if replaced.Version <= updated.Version {
				t.Fatalf("%s: version %d did not advance past %d", s.Name(), replaced.Version, updated.Version)
			}

			// A stale expected version is a conflict and changes nothing.
			stale := pack.Version
			conflict := map[string]string{"STALE": "write"}
			_, err = s.UpdatePack(t, pack.ID, &fibe.EnvPackUpdateParams{Env: &conflict, Version: &stale})
			requireEnvPackError(t, s.Name()+" stale version", err, envPackAttachmentInvalid, 409)
			after, err := s.GetPack(t, pack.ID)
			requireNoError(t, err)
			requireEnv(t, s.Name()+" after stale write", after.Env, next)

			// Explicit empty env is "no values" and differs from omitting env.
			empty := map[string]string{}
			cleared, err := s.UpdatePack(t, pack.ID, &fibe.EnvPackUpdateParams{Env: &empty, Version: &after.Version})
			requireNoError(t, err, s.Name()+" clear env")
			requireEnv(t, s.Name()+" clear env", cleared.Env, empty)
			renamedAgain := renamed + "-again"
			stillEmpty, err := s.UpdatePack(t, pack.ID, &fibe.EnvPackUpdateParams{Name: &renamedAgain})
			requireNoError(t, err)
			requireEnv(t, s.Name()+" omitted env after clear", stillEmpty.Env, empty)

			requireNoError(t, s.DeletePack(t, pack.ID), s.Name()+" delete")
			_, err = s.GetPack(t, pack.ID)
			requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
			if rows, err := s.ListPacks(t); err == nil {
				for _, row := range rows {
					if row.ID == pack.ID {
						t.Fatalf("%s: deleted pack %d is still listed", s.Name(), pack.ID)
					}
				}
			}
		})
	})
}

func TestEnvPacks_SDKRejectsNilAndInvalidValuesBeforeRequest(t *testing.T) {
	a := personalEnvPackActor(t)
	if _, err := a.client.EnvPacks.Create(ctx(), &fibe.EnvPackCreateParams{Name: uniqueName("nil-env")}); err == nil {
		t.Fatal("a nil env map must be rejected; use an empty map for no values")
	}
	if _, err := a.client.EnvPacks.Create(ctx(), &fibe.EnvPackCreateParams{Name: uniqueName("bad-key"), Env: map[string]string{"NOT-A-KEY": "x"}}); err == nil {
		t.Fatal("an invalid ENV key must be rejected")
	}
	dup := []fibe.EnvPackAttachmentInput{{EnvPackID: 1}, {EnvPackID: 1}}
	if _, err := a.client.EnvPacks.ReplaceAttachments(ctx(), "specs", 1, &fibe.EnvPackAttachmentsParams{Attachments: &dup}); err == nil {
		t.Fatal("duplicate pack IDs must be rejected before a request")
	}
	noNames := []string{}
	selected := []fibe.EnvPackAttachmentInput{{EnvPackID: 1, ServiceNames: &noNames}}
	if _, err := a.client.EnvPacks.ReplaceAttachments(ctx(), "specs", 1, &fibe.EnvPackAttachmentsParams{Attachments: &selected}); err == nil {
		t.Fatal("an empty selected-service list must be rejected before a request")
	}
}

func TestEnvPacks_ServerRejectsForgedOwnerAndBadValues(t *testing.T) {
	forEachEnvPackContext(t, func(t *testing.T, a *envPackActor) {
		pack := newEnvPack(t, a, "forged-owner", map[string]string{"KEEP": "me"})
		other := "Player"
		if a.ownerType == "Player" {
			other = "Team"
		}
		status, body := envPackRaw(t, a, "PATCH", fmt.Sprintf("/api/env_packs/%d", pack.ID),
			map[string]any{"env_pack": map[string]any{"owner_type": other, "owner_id": a.ownerID}})
		if status != 422 || rawErrorCode(body) != envPackAttachmentInvalid {
			t.Fatalf("owner transfer: status %d body %v, want 422 %s", status, body, envPackAttachmentInvalid)
		}
		for _, bad := range []any{1, true, map[string]any{"nested": "bad"}, []any{"nested"}} {
			status, _ = envPackRaw(t, a, "PATCH", fmt.Sprintf("/api/env_packs/%d", pack.ID),
				map[string]any{"env_pack": map[string]any{"env": map[string]any{"BAD": bad}}})
			if status != 422 {
				t.Fatalf("non-string ENV value %v: status %d, want 422", bad, status)
			}
		}
		got, err := a.client.EnvPacks.Get(ctx(), pack.ID)
		requireNoError(t, err)
		requireEnv(t, "after rejected writes", got.Env, map[string]string{"KEEP": "me"})
		status, body = envPackRaw(t, a, "POST", "/api/env_packs",
			map[string]any{"env_pack": map[string]any{"name": uniqueName("forged-create"), "env": map[string]any{}, "owner_type": other, "owner_id": a.ownerID}})
		if status != 422 || rawErrorCode(body) != envPackAttachmentInvalid {
			t.Fatalf("forged owner on create: status %d body %v", status, body)
		}
	})
}

func TestEnvPacks_ContextsAreIsolated(t *testing.T) {
	personal := personalEnvPackActor(t)
	company := companyEnvPackActor(t)
	personalPack := newEnvPack(t, personal, "isolated-personal", nil)
	companyPack := newEnvPack(t, company, "isolated-company", nil)
	for _, s := range []envPackSurface{sdkSurface{c: personal.client}, cliSurface{key: personal.key, domain: personal.domain}} {
		_, err := s.GetPack(t, companyPack.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	}
	for _, s := range []envPackSurface{sdkSurface{c: company.client}, cliSurface{key: company.key, domain: company.domain}} {
		_, err := s.GetPack(t, personalPack.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	}
}

// ------------------------------------------------------- attachment lifecycle

func TestEnvPacks_AttachmentLifecycle(t *testing.T) {
	forEachEnvPackContext(t, func(t *testing.T, a *envPackActor) {
		for _, kind := range []string{"specs", "playgrounds"} {
			kind := kind
			t.Run(kind, func(t *testing.T) {
				specID := newEnvPackSpec(t, a)
				targetID := specID
				if kind == "playgrounds" {
					targetID = newEnvPackPlayground(t, a, specID, envPackTargetHost(t, a), nil).ID
				}
				forEachSurface(t, a, func(t *testing.T, s envPackSurface) {
					exerciseAttachmentLifecycle(t, a, s, kind, targetID)
				})
			})
		}
	})
}

func exerciseAttachmentLifecycle(t *testing.T, a *envPackActor, s envPackSurface, kind string, id int64) {
	secret := uniqueName("lifecycle-secret")
	packA := newEnvPack(t, a, "lifecycle-a", map[string]string{"TOKEN": secret})
	packB := newEnvPack(t, a, "lifecycle-b", map[string]string{"B": "b"})
	name := s.Name()

	// Start empty; an explicit [] detaches everything.
	cleared, err := s.Replace(t, kind, id, emptyAttachments())
	requireNoError(t, err, name+" clear")
	requireAttachmentOrder(t, name+" clear", cleared)

	// set: ordered, all services for A and web only for B; nil service_names means all.
	set, err := s.Replace(t, kind, id, &[]fibe.EnvPackAttachmentInput{{EnvPackID: packA.ID}, attachNamed(packB.ID, "web")})
	requireNoError(t, err, name+" set")
	requireAttachmentOrder(t, name+" set", set, packA.ID, packB.ID)
	requireNames(t, name+" set", attachmentFor(t, set, packA.ID))
	requireNames(t, name+" set", attachmentFor(t, set, packB.ID), "web")
	for _, row := range set.EnvPackAttachments {
		if row.GrantingSubject == "" || row.GrantedAt == nil {
			t.Fatalf("%s: attachment %d has no recorded grant: %+v", name, row.EnvPackID, row)
		}
	}
	requireNoLeak(t, name+" set response", set, secret)
	grantA := attachmentFor(t, set, packA.ID)
	grantB := attachmentFor(t, set, packB.ID)

	// nil (omitted) preserves the list; [] is the only way to detach.
	kept, err := s.Replace(t, kind, id, nil)
	requireNoError(t, err, name+" omitted selection")
	requireAttachmentOrder(t, name+" omitted selection preserves", kept, packA.ID, packB.ID)
	requireSameGrant(t, name, grantA, attachmentFor(t, kept, packA.ID))

	// reorder keeps every unchanged grant intact.
	reordered, err := s.Reorder(t, kind, id, []int64{packB.ID, packA.ID})
	requireNoError(t, err, name+" reorder")
	requireAttachmentOrder(t, name+" reorder", reordered, packB.ID, packA.ID)
	requireSameGrant(t, name+" reorder A", grantA, attachmentFor(t, reordered, packA.ID))
	requireSameGrant(t, name+" reorder B", grantB, attachmentFor(t, reordered, packB.ID))
	if _, err := s.Reorder(t, kind, id, []int64{packB.ID}); err == nil {
		t.Fatalf("%s: a partial order must be rejected", name)
	}

	// retarget a single pack to selected services and back to all services; the other grant stays.
	retargeted, err := s.Retarget(t, kind, id, packB.ID, &[]string{"db"})
	requireNoError(t, err, name+" retarget")
	requireNames(t, name+" retarget", attachmentFor(t, retargeted, packB.ID), "db")
	requireSameGrant(t, name+" retarget leaves A", grantA, attachmentFor(t, retargeted, packA.ID))
	widened, err := s.Retarget(t, kind, id, packB.ID, nil)
	requireNoError(t, err, name+" retarget all services")
	requireNames(t, name+" retarget all services", attachmentFor(t, widened, packB.ID))
	requireAttachmentOrder(t, name+" retarget keeps order", widened, packB.ID, packA.ID)

	// renew re-grants exactly the selected pack.
	beforeRenew := attachmentFor(t, widened, packB.ID)
	renewed, err := s.Renew(t, kind, id, packB.ID)
	requireNoError(t, err, name+" renew")
	afterRenew := attachmentFor(t, renewed, packB.ID)
	if afterRenew.GrantedAt == nil || beforeRenew.GrantedAt == nil || afterRenew.GrantedAt.Before(*beforeRenew.GrantedAt) {
		t.Fatalf("%s: renew moved the grant backwards: %v -> %v", name, beforeRenew.GrantedAt, afterRenew.GrantedAt)
	}
	requireSameGrant(t, name+" renew leaves A", grantA, attachmentFor(t, renewed, packA.ID))
	requireAttachmentOrder(t, name+" renew keeps order", renewed, packB.ID, packA.ID)

	// detach removes only the selected pack; detaching it again is an error.
	detached, err := s.Detach(t, kind, id, packA.ID)
	requireNoError(t, err, name+" detach")
	requireAttachmentOrder(t, name+" detach", detached, packB.ID)
	if _, err := s.Detach(t, kind, id, packA.ID); err == nil {
		t.Fatalf("%s: detaching an unattached pack must fail", name)
	}

	// provenance read matches and never carries pack values.
	read, err := s.Attachments(t, kind, id)
	requireNoError(t, err, name+" read")
	requireAttachmentOrder(t, name+" read", read, packB.ID)
	requireNoLeak(t, name+" provenance", read, secret)

	// [] detaches the rest; omitting afterwards keeps it empty.
	emptied, err := s.Replace(t, kind, id, emptyAttachments())
	requireNoError(t, err, name+" detach all")
	requireAttachmentOrder(t, name+" detach all", emptied)
	still, err := s.Replace(t, kind, id, nil)
	requireNoError(t, err, name+" omitted after detach all")
	requireAttachmentOrder(t, name+" omitted after detach all", still)
}

// ----------------------------------------------------------- stable error codes

func TestEnvPacks_StableErrorCodes(t *testing.T) {
	forEachEnvPackContext(t, func(t *testing.T, a *envPackActor) {
		forEachSurface(t, a, func(t *testing.T, s envPackSurface) {
			exerciseErrorCodes(t, a, s)
		})
	})
}

func exerciseErrorCodes(t *testing.T, a *envPackActor, s envPackSurface) {
	secret := uniqueName("error-secret")
	pack := newEnvPack(t, a, "errors", map[string]string{"TOKEN": secret})
	specID := newEnvPackSpec(t, a)
	name := s.Name()

	baseline, err := a.client.EnvPacks.ReplaceAttachments(ctx(), "specs", specID,
		&fibe.EnvPackAttachmentsParams{Attachments: &[]fibe.EnvPackAttachmentInput{attachNamed(pack.ID, "web")}})
	requireNoError(t, err)
	grant := attachmentFor(t, baseline, pack.ID)

	unchanged := func(step string) {
		t.Helper()
		now, err := a.client.EnvPacks.Attachments(ctx(), "specs", specID)
		requireNoError(t, err)
		requireAttachmentOrder(t, name+" "+step+" leaves list", now, pack.ID)
		requireSameGrant(t, name+" "+step, grant, attachmentFor(t, now, pack.ID))
	}

	// env_pack_service_missing: the selected service does not exist on the target.
	_, err = s.Replace(t, "specs", specID, &[]fibe.EnvPackAttachmentInput{attachNamed(pack.ID, "missing-exact-target")})
	apiErr := requireEnvPackError(t, name+" service missing", err, envPackServiceMissing, 422)
	if names, _ := apiErr.Details["service_names"].([]any); len(names) != 1 || names[0] != "missing-exact-target" {
		t.Fatalf("%s: service_missing details %v", name, apiErr.Details)
	}
	requireNoLeak(t, name+" service_missing error", apiErr, secret)
	unchanged("service_missing")

	// env_pack_unavailable: a reference that does not exist.
	_, err = s.Replace(t, "specs", specID, &[]fibe.EnvPackAttachmentInput{{EnvPackID: pack.ID}, {EnvPackID: missingEnvPackID}})
	apiErr = requireEnvPackError(t, name+" missing pack", err, envPackUnavailable, 422)
	requireNoLeak(t, name+" unavailable error", apiErr, secret)
	unchanged("unavailable (missing)")

	// env_pack_unavailable: a deleted pack, even though it is already on the list.
	doomed := newEnvPack(t, a, "errors-deleted", nil)
	requireNoError(t, a.client.EnvPacks.Delete(ctx(), doomed.ID))
	_, err = s.Replace(t, "specs", specID, &[]fibe.EnvPackAttachmentInput{{EnvPackID: doomed.ID}})
	requireEnvPackError(t, name+" deleted pack", err, envPackUnavailable, 422)
	unchanged("unavailable (deleted)")

	// env_pack_attachment_invalid: a stale pack version is a 409 conflict with the same stable code.
	stale := pack.Version + 1000
	other := map[string]string{"STALE": "write"}
	_, err = s.UpdatePack(t, pack.ID, &fibe.EnvPackUpdateParams{Env: &other, Version: &stale})
	requireEnvPackError(t, name+" stale version", err, envPackAttachmentInvalid, 409)
	got, err := a.client.EnvPacks.Get(ctx(), pack.ID)
	requireNoError(t, err)
	requireEnv(t, name+" env after conflict", got.Env, map[string]string{"TOKEN": secret})
}

// The CLI and MCP surfaces validate locally, so duplicate, forged and empty
// selections are sent raw to prove the server enforces the same contract.
func TestEnvPacks_ServerRejectsInvalidAttachmentLists(t *testing.T) {
	forEachEnvPackContext(t, func(t *testing.T, a *envPackActor) {
		pack := newEnvPack(t, a, "raw-invalid", nil)
		specID := newEnvPackSpec(t, a)
		path := fmt.Sprintf("/api/specs/%d/env_pack_attachments", specID)
		valid, err := a.client.EnvPacks.ReplaceAttachments(ctx(), "specs", specID, &fibe.EnvPackAttachmentsParams{Attachments: attach(pack.ID)})
		requireNoError(t, err)
		grant := attachmentFor(t, valid, pack.ID)

		for label, rows := range map[string]any{
			"duplicate IDs":        []any{map[string]any{"env_pack_id": pack.ID}, map[string]any{"env_pack_id": pack.ID}},
			"forged grantor":       []any{map[string]any{"env_pack_id": pack.ID, "grantor_id": 1, "grantor_type": "Player"}},
			"forged position":      []any{map[string]any{"env_pack_id": pack.ID, "position": 7}},
			"empty service_names":  []any{map[string]any{"env_pack_id": pack.ID, "service_names": []any{}}},
			"duplicate services":   []any{map[string]any{"env_pack_id": pack.ID, "service_names": []any{"web", "web"}}},
			"non-positive ID":      []any{map[string]any{"env_pack_id": 0}},
			"string service entry": []any{map[string]any{"env_pack_id": pack.ID, "service_names": []any{7}}},
			"not a list":           map[string]any{"env_pack_id": pack.ID},
		} {
			status, body := envPackRaw(t, a, "PATCH", path, map[string]any{"env_pack_attachments": rows})
			if status != 422 || rawErrorCode(body) != envPackAttachmentInvalid {
				t.Errorf("%s: status %d code %q, want 422 %s (body %v)", label, status, rawErrorCode(body), envPackAttachmentInvalid, body)
			}
		}
		after, err := a.client.EnvPacks.Attachments(ctx(), "specs", specID)
		requireNoError(t, err)
		requireAttachmentOrder(t, "after rejected lists", after, pack.ID)
		requireSameGrant(t, "after rejected lists", grant, attachmentFor(t, after, pack.ID))
	})
}

// ------------------------------------------------------------ delegated source

// A grant recorded by an earlier authority must survive a later manager who has
// target authority but no authority over the pack, and that manager can neither
// acquire nor renew the pack.
func TestEnvPacks_DelegatedSourceSurvivesTargetOnlyManager(t *testing.T) {
	a := personalEnvPackActor(t)
	owner := sdkSurface{c: a.client}
	packA := newEnvPack(t, a, "delegated-a", map[string]string{"TOKEN": uniqueName("delegated-secret")})
	packB := newEnvPack(t, a, "delegated-b", nil)
	specID := newEnvPackSpec(t, a)

	granted, err := owner.Replace(t, "specs", specID, &[]fibe.EnvPackAttachmentInput{{EnvPackID: packA.ID}, {EnvPackID: packB.ID}})
	requireNoError(t, err)
	grantA, grantB := attachmentFor(t, granted, packA.ID), attachmentFor(t, granted, packB.ID)

	scoped, err := a.client.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
		Label:  uniqueName("target-only"),
		Scopes: []string{"specs:read", "specs:write"}, // no env_packs:read
	})
	requireNoError(t, err, "create target-only key")
	if scoped.Token == nil || scoped.ID == nil {
		t.Fatal("target-only key response has no token")
	}
	t.Cleanup(func() { _ = a.client.APIKeys.Delete(ctx(), *scoped.ID) })
	delegate := &envPackActor{label: "personal", key: *scoped.Token, domain: a.domain, client: a.client.WithKey(*scoped.Token),
		ownerType: a.ownerType, ownerID: a.ownerID}

	forEachSurface(t, delegate, func(t *testing.T, s envPackSurface) {
		name := s.Name()
		// It cannot acquire a new reference...
		newPack := newEnvPack(t, a, "delegated-new", nil)
		_, err := s.Replace(t, "specs", specID, &[]fibe.EnvPackAttachmentInput{{EnvPackID: packA.ID}, {EnvPackID: packB.ID}, {EnvPackID: newPack.ID}})
		requireEnvPackError(t, name+" acquire without source authority", err, envPackUnavailable, 422)
		// ...nor renew an existing one.
		_, err = s.Renew(t, "specs", specID, packA.ID)
		requireEnvPackError(t, name+" renew without source authority", err, envPackUnavailable, 422)
		now, err := owner.Attachments(t, "specs", specID)
		requireNoError(t, err)
		requireAttachmentOrder(t, name+" after refused writes", now, packA.ID, packB.ID)

		// It can preserve and reorder what is already granted, without re-granting.
		same, err := s.Replace(t, "specs", specID, &[]fibe.EnvPackAttachmentInput{{EnvPackID: packA.ID}, {EnvPackID: packB.ID}})
		requireNoError(t, err, name+" preserve existing selection")
		requireSameGrant(t, name+" preserve A", grantA, attachmentFor(t, same, packA.ID))
		requireSameGrant(t, name+" preserve B", grantB, attachmentFor(t, same, packB.ID))
		swapped, err := s.Reorder(t, "specs", specID, []int64{packB.ID, packA.ID})
		requireNoError(t, err, name+" reorder existing selection")
		requireAttachmentOrder(t, name+" reorder", swapped, packB.ID, packA.ID)
		requireSameGrant(t, name+" reorder A", grantA, attachmentFor(t, swapped, packA.ID))
		requireSameGrant(t, name+" reorder B", grantB, attachmentFor(t, swapped, packB.ID))
		restored, err := s.Reorder(t, "specs", specID, []int64{packA.ID, packB.ID})
		requireNoError(t, err)
		requireAttachmentOrder(t, name+" restore order", restored, packA.ID, packB.ID)
	})
}

// servicePrincipal is a Company service principal with its own credential.
type servicePrincipal struct {
	id      int64
	subject string
	client  *fibe.Client
	revoked bool
	company *envPackActor
}

func newServicePrincipal(t *testing.T, company *envPackActor) *servicePrincipal {
	t.Helper()
	status, body := envPackRaw(t, company, "POST", "/api/team_service_principals",
		map[string]any{"service_principal": map[string]any{"name": uniqueName("envpack-principal")}})
	if status != 201 {
		t.Fatalf("create service principal: status %d body %v", status, body)
	}
	id, ok := body["id"].(float64)
	if !ok {
		t.Fatalf("service principal response has no id: %v", body)
	}
	p := &servicePrincipal{id: int64(id), company: company}
	p.subject = "TeamServicePrincipal:" + strconv.FormatInt(p.id, 10)
	t.Cleanup(func() { p.revoke(t) })
	key, err := company.client.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
		Label:         uniqueName("envpack-principal-key"),
		PrincipalType: "TeamServicePrincipal", PrincipalID: &p.id,
		Scopes: []string{"env_packs:read", "specs:read", "specs:write"},
	})
	requireNoError(t, err, "issue service principal credential")
	if key.Token == nil {
		t.Fatal("service principal credential has no token")
	}
	p.client = integrationClient(*key.Token, company.domain, 2)
	return p
}

func (p *servicePrincipal) revoke(t *testing.T) {
	t.Helper()
	if p.revoked {
		return
	}
	p.revoked = true
	status, body := envPackRaw(t, p.company, "DELETE", "/api/team_service_principals/"+strconv.FormatInt(p.id, 10), nil)
	if status != 204 && status != 404 {
		t.Errorf("revoke service principal %d: status %d body %v", p.id, status, body)
	}
}

// A Company Spec reference granted by a service principal is a delegated grant.
// While the principal is active an administrator can keep it as is. After the
// principal is revoked, a write that would keep the stale grant fails with
// env_pack_grant_invalid until the administrator renews it under their own
// authority. Each surface gets its own principal because renewal consumes the
// stale state. Revoking a principal revises the Company's authority, so this
// test is kept to Specs, which have no queued runtime operations.
func TestEnvPacks_CompanyDelegatedGrantInvalidAfterGrantorRevoked(t *testing.T) {
	company := companyEnvPackActor(t)
	forEachSurface(t, company, func(t *testing.T, s envPackSurface) {
		name := s.Name()
		pack := newEnvPack(t, company, "grant-invalid", map[string]string{"TOKEN": uniqueName("grant-secret")})
		specID := newEnvPackSpec(t, company)
		principal := newServicePrincipal(t, company)

		granted, err := principal.client.EnvPacks.ReplaceAttachments(ctx(), "specs", specID, &fibe.EnvPackAttachmentsParams{Attachments: attach(pack.ID)})
		requireNoError(t, err, "principal attaches Company pack")
		delegated := attachmentFor(t, granted, pack.ID)
		if delegated.GrantingSubject != principal.subject {
			t.Fatalf("%s: grantor %q, want %q", name, delegated.GrantingSubject, principal.subject)
		}

		// Active grantor: the same selection is kept without re-granting.
		kept, err := s.Replace(t, "specs", specID, attach(pack.ID))
		requireNoError(t, err, name+" preserve delegated grant")
		requireSameGrant(t, name+" preserve delegated grant", delegated, attachmentFor(t, kept, pack.ID))

		principal.revoke(t)

		_, err = s.Replace(t, "specs", specID, attach(pack.ID))
		apiErr := requireEnvPackError(t, name+" stale delegated grant", err, envPackGrantInvalid, 422)
		requireNoLeak(t, name+" grant_invalid error", apiErr, "grant-secret")
		after, err := company.client.EnvPacks.Attachments(ctx(), "specs", specID)
		requireNoError(t, err)
		requireAttachmentOrder(t, name+" after grant_invalid", after, pack.ID)
		requireSameGrant(t, name+" after grant_invalid", delegated, attachmentFor(t, after, pack.ID))

		renewed, err := s.Renew(t, "specs", specID, pack.ID)
		requireNoError(t, err, name+" renew stale delegated grant")
		fresh := attachmentFor(t, renewed, pack.ID)
		if fresh.GrantingSubject == principal.subject {
			t.Fatalf("%s: renew kept the revoked grantor %s", name, principal.subject)
		}
		again, err := s.Replace(t, "specs", specID, attach(pack.ID))
		requireNoError(t, err, name+" selection after renew")
		requireSameGrant(t, name+" after renew", fresh, attachmentFor(t, again, pack.ID))
	})
}

// ------------------------------------------------------------ launch selection

func TestEnvPacks_LaunchWithAttachments(t *testing.T) {
	a := personalEnvPackActor(t)
	host := envPackHost(t)
	forEachSurface(t, a, func(t *testing.T, s envPackSurface) {
		name := s.Name()
		packA := newEnvPack(t, a, "launch-a", map[string]string{"A": "a"})
		packB := newEnvPack(t, a, "launch-b", map[string]string{"B": "b"})

		playgroundAttachments := func(id int64) *fibe.EnvPackTarget {
			t.Helper()
			target, err := a.client.EnvPacks.Attachments(ctx(), "playgrounds", id)
			requireNoError(t, err)
			return target
		}
		launch := func(step string, rows *[]fibe.EnvPackAttachmentInput) *fibe.LaunchResult {
			t.Helper()
			result, err := s.LaunchCompose(t, uniqueName("envpack-launch-"+name+"-"+step), host, minimalComposeYAML(), rows)
			requireNoError(t, err, name+" launch "+step)
			if result.PlaygroundID == 0 || result.SpecID == 0 {
				t.Fatalf("%s launch %s: result %+v has no Spec and Playground", name, step, result)
			}
			t.Cleanup(func() { _ = a.client.Specs.Delete(ctx(), result.SpecID) }) // runs after the Playground is deleted
			cleanupPlayground(t, a, result.PlaygroundID)
			return result
		}

		explicit := launch("explicit", &[]fibe.EnvPackAttachmentInput{{EnvPackID: packA.ID}, attachNamed(packB.ID, "web")})
		created := playgroundAttachments(explicit.PlaygroundID)
		requireAttachmentOrder(t, name+" launch explicit", created, packA.ID, packB.ID)
		requireNames(t, name+" launch explicit", attachmentFor(t, created, packB.ID), "web")
		// The recipient selection belongs to the Playground; the launched Spec stays reference-free.
		specTarget, err := a.client.EnvPacks.Attachments(ctx(), "specs", explicit.SpecID)
		requireNoError(t, err)
		requireAttachmentOrder(t, name+" launched Spec", specTarget)

		optOut := launch("opt-out", emptyAttachments())
		requireAttachmentOrder(t, name+" launch []", playgroundAttachments(optOut.PlaygroundID))

		inherited := launch("inherit", nil)
		requireAttachmentOrder(t, name+" launch nil", playgroundAttachments(inherited.PlaygroundID))

		// From an existing Spec: nil inherits its defaults, [] opts out, a list replaces them.
		specID := newEnvPackSpec(t, a)
		spec, err := a.client.Specs.Get(ctx(), specID)
		requireNoError(t, err)
		_, err = a.client.EnvPacks.ReplaceAttachments(ctx(), "specs", specID, &fibe.EnvPackAttachmentsParams{Attachments: attach(packA.ID)})
		requireNoError(t, err)
		fromSpec := func(step string, rows *[]fibe.EnvPackAttachmentInput) *fibe.Playground {
			t.Helper()
			pg, err := s.LaunchSpec(t, spec.Name, uniqueName("envpack-spec-"+name+"-"+step), host, rows)
			requireNoError(t, err, name+" spec launch "+step)
			cleanupPlayground(t, a, pg.ID)
			return pg
		}
		requireAttachmentOrder(t, name+" spec launch nil", playgroundAttachments(fromSpec("inherit", nil).ID), packA.ID)
		requireAttachmentOrder(t, name+" spec launch []", playgroundAttachments(fromSpec("opt-out", emptyAttachments()).ID))
		requireAttachmentOrder(t, name+" spec launch [B]", playgroundAttachments(fromSpec("replace", attach(packB.ID)).ID), packB.ID)
		specAfter, err := a.client.EnvPacks.Attachments(ctx(), "specs", specID)
		requireNoError(t, err)
		requireAttachmentOrder(t, name+" Spec defaults untouched by launches", specAfter, packA.ID)

		// A reference that cannot be resolved fails the launch before anything is created.
		_, err = s.LaunchSpec(t, spec.Name, uniqueName("envpack-bad-"+name), host, attach(missingEnvPackID))
		requireEnvPackError(t, name+" launch unavailable pack", err, envPackUnavailable, 422)
	})
}

// ------------------------------------------------------------------ safe rerun

func TestEnvPacks_SafeRerun(t *testing.T) {
	a := personalEnvPackActor(t)
	host := envPackHost(t)
	forEachSurface(t, a, func(t *testing.T, s envPackSurface) {
		secret := uniqueName("rerun-secret")
		packA := newEnvPack(t, a, "rerun-a", map[string]string{"TOKEN": secret})
		packB := newEnvPack(t, a, "rerun-b", map[string]string{"B": "b"})
		target := func(id int64) *fibe.EnvPackTarget {
			t.Helper()
			out, err := a.client.EnvPacks.Attachments(ctx(), "playgrounds", id)
			requireNoError(t, err)
			return out
		}
		count := func(q string, job bool) int {
			t.Helper()
			list, err := a.client.Playgrounds.List(ctx(), &fibe.PlaygroundListParams{Q: q, JobMode: &job, PerPage: 100})
			requireNoError(t, err)
			return len(list.Data)
		}

		t.Run("playground", func(t *testing.T) {
			specID := newEnvPackSpec(t, a)
			_, err := a.client.EnvPacks.ReplaceAttachments(ctx(), "specs", specID, &fibe.EnvPackAttachmentsParams{
				Attachments: &[]fibe.EnvPackAttachmentInput{{EnvPackID: packA.ID}, attachNamed(packB.ID, "web")}})
			requireNoError(t, err)
			source := newEnvPackPlayground(t, a, specID, host, nil)
			before := target(source.ID)
			requireAttachmentOrder(t, "source inherits Spec defaults", before, packA.ID, packB.ID)

			sameAsSource := func(step string, got *fibe.EnvPackTarget) {
				t.Helper()
				requireAttachmentOrder(t, s.Name()+" "+step, got, packA.ID, packB.ID)
				requireNames(t, s.Name()+" "+step, attachmentFor(t, got, packB.ID), "web")
				for _, id := range []int64{packA.ID, packB.ID} {
					requireSameGrant(t, s.Name()+" "+step+" keeps the source grant", attachmentFor(t, before, id), attachmentFor(t, got, id))
				}
			}
			sourceUnchanged := func(step string) {
				t.Helper()
				sameAsSource(step+": source", target(source.ID))
			}

			// Omitted selection: same ordered references and the source's recorded grants.
			wanted := uniqueName("envpack-rerun-copy")
			inherited, err := s.RerunPlayground(t, source.ID, wanted, nil)
			requireNoError(t, err, s.Name()+" rerun inherit")
			cleanupPlayground(t, a, inherited.ID)
			if inherited.ID == source.ID || inherited.Name != wanted {
				t.Fatalf("%s: rerun result %d/%q, want a new Playground named %q", s.Name(), inherited.ID, inherited.Name, wanted)
			}
			requireNoLeak(t, s.Name()+" rerun response", inherited, secret)
			inheritedTarget := target(inherited.ID)
			sameAsSource("rerun inherit", inheritedTarget)
			sourceUnchanged("after inherit")

			// Explicit [] detaches every pack on the new Playground only.
			optOut, err := s.RerunPlayground(t, source.ID, "", emptyAttachments())
			requireNoError(t, err, s.Name()+" rerun opt-out")
			cleanupPlayground(t, a, optOut.ID)
			requireAttachmentOrder(t, s.Name()+" rerun []", target(optOut.ID))
			sourceUnchanged("after opt-out")

			// Explicit list replaces the references.
			picked, err := s.RerunPlayground(t, source.ID, "", attach(packB.ID))
			requireNoError(t, err, s.Name()+" rerun explicit")
			cleanupPlayground(t, a, picked.ID)
			pickedTarget := target(picked.ID)
			requireAttachmentOrder(t, s.Name()+" rerun [B]", pickedTarget, packB.ID)
			requireNames(t, s.Name()+" rerun [B]", attachmentFor(t, pickedTarget, packB.ID))

			// Later changes to the source never reach a Playground already created.
			_, err = a.client.EnvPacks.ReplaceAttachments(ctx(), "playgrounds", source.ID, &fibe.EnvPackAttachmentsParams{Attachments: attach(packB.ID)})
			requireNoError(t, err)
			sameAsSource("rerun after source change", target(inherited.ID))
			_, err = a.client.EnvPacks.ReplaceAttachments(ctx(), "playgrounds", source.ID, &fibe.EnvPackAttachmentsParams{
				Attachments: &[]fibe.EnvPackAttachmentInput{{EnvPackID: packA.ID}, attachNamed(packB.ID, "web")}})
			requireNoError(t, err)

			// A reference that cannot resolve fails with the stable code and leaves nothing behind.
			existing := count(source.Name, false)
			_, err = s.RerunPlayground(t, source.ID, "", attach(missingEnvPackID))
			requireEnvPackError(t, s.Name()+" rerun unavailable pack", err, envPackUnavailable, 422)
			if now := count(source.Name, false); now != existing {
				t.Fatalf("%s: failed rerun left a partial Playground (%d -> %d)", s.Name(), existing, now)
			}
			_, err = s.RerunPlayground(t, source.ID, "", &[]fibe.EnvPackAttachmentInput{attachNamed(packB.ID, "missing-exact-target")})
			requireEnvPackError(t, s.Name()+" rerun missing service", err, envPackServiceMissing, 422)
			if now := count(source.Name, false); now != existing {
				t.Fatalf("%s: failed rerun left a partial Playground (%d -> %d)", s.Name(), existing, now)
			}

			if _, ok := s.(sdkSurface); ok {
				// A retried request with the same idempotency key replays one Playground.
				key := fibe.NewIdempotencyKey()
				first, err := a.client.Playgrounds.Rerun(fibe.WithIdempotencyKey(ctx(), key), source.ID, &fibe.PlaygroundRerunParams{})
				requireNoError(t, err, "idempotent rerun")
				cleanupPlayground(t, a, first.ID)
				second, err := a.client.Playgrounds.Rerun(fibe.WithIdempotencyKey(ctx(), key), source.ID, &fibe.PlaygroundRerunParams{})
				requireNoError(t, err, "idempotent rerun replay")
				if second.ID != first.ID {
					t.Fatalf("replayed rerun created Playground %d, want the original %d", second.ID, first.ID)
				}
				// Distinct explicit reruns remain distinct.
				third, err := a.client.Playgrounds.Rerun(ctx(), source.ID, &fibe.PlaygroundRerunParams{})
				requireNoError(t, err)
				cleanupPlayground(t, a, third.ID)
				if third.ID == first.ID {
					t.Fatalf("a new rerun reused Playground %d", first.ID)
				}
			}
		})

		t.Run("task", func(t *testing.T) {
			spec := newEnvPackJobSpec(t, a)
			_, err := a.client.EnvPacks.ReplaceAttachments(ctx(), "specs", *spec.ID, &fibe.EnvPackAttachmentsParams{
				Attachments: &[]fibe.EnvPackAttachmentInput{{EnvPackID: packA.ID}, attachNamed(packB.ID, "worker")}})
			requireNoError(t, err)
			task, err := a.client.Tasks.Trigger(ctx(), &fibe.TaskTriggerParams{SpecID: *spec.ID, HostID: &host})
			requireNoError(t, err, "trigger source Task")
			t.Cleanup(func() { _ = a.client.Tasks.Delete(ctx(), task.ID) })
			before := target(task.ID)
			requireAttachmentOrder(t, "Task inherits Spec defaults", before, packA.ID, packB.ID)

			rerun, err := s.RerunTask(t, strconv.FormatInt(task.ID, 10), nil)
			requireNoError(t, err, s.Name()+" task rerun inherit")
			t.Cleanup(func() { _ = a.client.Tasks.Delete(ctx(), rerun.ID) })
			if rerun.ID == task.ID || !rerun.JobMode {
				t.Fatalf("%s: task rerun result %d job_mode=%v", s.Name(), rerun.ID, rerun.JobMode)
			}
			got := target(rerun.ID)
			requireAttachmentOrder(t, s.Name()+" task rerun inherit", got, packA.ID, packB.ID)
			requireNames(t, s.Name()+" task rerun inherit", attachmentFor(t, got, packB.ID), "worker")
			for _, id := range []int64{packA.ID, packB.ID} {
				requireSameGrant(t, s.Name()+" task rerun keeps the source grant", attachmentFor(t, before, id), attachmentFor(t, got, id))
			}
			requireNoLeak(t, s.Name()+" task rerun response", rerun, secret)

			// By name, with an explicit [] opt-out.
			optOut, err := s.RerunTask(t, task.Name, emptyAttachments())
			requireNoError(t, err, s.Name()+" task rerun opt-out")
			t.Cleanup(func() { _ = a.client.Tasks.Delete(ctx(), optOut.ID) })
			requireAttachmentOrder(t, s.Name()+" task rerun []", target(optOut.ID))
			requireAttachmentOrder(t, s.Name()+" source Task unchanged", target(task.ID), packA.ID, packB.ID)

			existing := count(task.Name, true)
			_, err = s.RerunTask(t, strconv.FormatInt(task.ID, 10), attach(missingEnvPackID))
			requireEnvPackError(t, s.Name()+" task rerun unavailable pack", err, envPackUnavailable, 422)
			if now := count(task.Name, true); now != existing {
				t.Fatalf("%s: failed Task rerun left a partial Task (%d -> %d)", s.Name(), existing, now)
			}
		})
	})
}

// =============================================================== support

// The four stable ENV-pack error codes (EnvPacks::Error in the Rails app).
const (
	envPackAttachmentInvalid = "env_pack_attachment_invalid"
	envPackUnavailable       = "env_pack_unavailable"
	envPackGrantInvalid      = "env_pack_grant_invalid"
	envPackServiceMissing    = "env_pack_service_missing"
)

// envPackActor is one authenticated owner context.
type envPackActor struct {
	label     string // "personal" or "company"
	key       string
	domain    string
	client    *fibe.Client
	ownerType string
	ownerID   int64
	principal string // "Player:<id>" style subject for this credential
}

func envPackDomain() string {
	if domain := os.Getenv("FIBE_DOMAIN"); domain != "" {
		return domain
	}
	return "localhost:3000"
}

func personalEnvPackActor(t *testing.T) *envPackActor {
	t.Helper()
	c := userClient(t)
	me, err := c.APIKeys.Me(ctx())
	requireNoError(t, err, "resolve Personal credential")
	if me.OwnerType != "Player" {
		t.Fatalf("FIBE_API_KEY must be a Personal credential, got owner_type %q", me.OwnerType)
	}
	return &envPackActor{label: "personal", key: os.Getenv("FIBE_API_KEY"), domain: envPackDomain(), client: c,
		ownerType: me.OwnerType, ownerID: me.OwnerID, principal: fmt.Sprintf("%s:%d", me.PrincipalType, me.PrincipalID)}
}

func companyEnvPackActor(t *testing.T) *envPackActor {
	t.Helper()
	userClient(t)
	key := strings.TrimSpace(os.Getenv("RUNTIME_API_KEY"))
	if key == "" {
		reason := "Company context needs RUNTIME_API_KEY, RUNTIME_OWNER_TYPE=Team and RUNTIME_OWNER_ID; " +
			"the app-go-tests lane bootstraps mode=sdk, which does not export them (only the Playwright bootstrap does). " +
			"Export them, or set FIBE_REQUIRE_COMPANY_ENV_PACKS=1 to fail instead of skipping"
		if envBool("FIBE_REQUIRE_COMPANY_ENV_PACKS") {
			t.Fatal(reason)
		}
		t.Skip(reason)
	}
	c := integrationClient(key, envPackDomain(), 2)
	me, err := c.APIKeys.Me(ctx())
	requireNoError(t, err, "resolve Company credential")
	if me.OwnerType != "Team" {
		t.Fatalf("RUNTIME_API_KEY must be a Company credential, got owner_type %q", me.OwnerType)
	}
	if want := strings.TrimSpace(os.Getenv("RUNTIME_OWNER_TYPE")); want != "" && want != me.OwnerType {
		t.Fatalf("RUNTIME_OWNER_TYPE=%q does not match the credential's %q", want, me.OwnerType)
	}
	if raw := strings.TrimSpace(os.Getenv("RUNTIME_OWNER_ID")); raw != "" {
		want, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || want != me.OwnerID {
			t.Fatalf("RUNTIME_OWNER_ID=%q does not match the credential's owner %d", raw, me.OwnerID)
		}
	}
	return &envPackActor{label: "company", key: key, domain: envPackDomain(), client: c,
		ownerType: me.OwnerType, ownerID: me.OwnerID, principal: fmt.Sprintf("%s:%d", me.PrincipalType, me.PrincipalID)}
}

// envPackHost returns the funded lane Host used for Playground and Task targets.
// Skipping is limited to runs with no Host at all (no bootstrap, no FIBE_TEST_HOST_ID).
func envPackHost(t *testing.T) int64 {
	t.Helper()
	id := testHostID(t)
	if id == 0 {
		if envBool("FIBE_E2E_BOOTSTRAP") || envBool("SDK_E2E_BOOTSTRAP") {
			t.Fatal("the SDK e2e bootstrap must provide a funded Host (FIBE_TEST_HOST_ID)")
		}
		t.Skip("no funded Host available: set FIBE_TEST_HOST_ID (provided by the SDK e2e bootstrap)")
	}
	return id
}

// ---------------------------------------------------------------- fixtures

func newEnvPack(t *testing.T, a *envPackActor, prefix string, env map[string]string) *fibe.EnvPack {
	t.Helper()
	if env == nil {
		env = map[string]string{}
	}
	pack, err := a.client.EnvPacks.Create(ctx(), &fibe.EnvPackCreateParams{Name: uniqueName(prefix), Env: env})
	requireNoError(t, err, "create ENV pack fixture")
	t.Cleanup(func() { _ = a.client.EnvPacks.Delete(ctx(), pack.ID) })
	return pack
}

func newEnvPackSpec(t *testing.T, a *envPackActor) int64 {
	t.Helper()
	spec := seedSpec(t, a.client, func(p *fibe.SpecCreateParams) {
		p.BaseComposeYAML = "services:\n  web:\n    image: nginx:alpine\n  db:\n    image: postgres:16-alpine\n  cache:\n    image: redis:7-alpine\n"
	})
	return *spec.ID
}

func newEnvPackJobSpec(t *testing.T, a *envPackActor) *fibe.Spec {
	t.Helper()
	jobMode := true
	return seedSpec(t, a.client, func(p *fibe.SpecCreateParams) {
		p.JobMode = &jobMode
		p.BaseComposeYAML = jobComposeYAML()
		p.Services = []fibe.SpecServiceDef{jobWatchedService("worker")}
	})
}

func newEnvPackPlayground(t *testing.T, a *envPackActor, specID, hostID int64, rows *[]fibe.EnvPackAttachmentInput) *fibe.Playground {
	t.Helper()
	pg, err := a.client.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name: uniqueName("envpack-pg"), SpecID: specID, HostID: &hostID, EnvPackAttachments: rows,
	})
	requireNoError(t, err, "create Playground fixture")
	t.Cleanup(func() { _ = a.client.Playgrounds.Delete(ctx(), pg.ID) })
	return pg
}

func cleanupPlayground(t *testing.T, a *envPackActor, id int64) {
	t.Helper()
	t.Cleanup(func() { _ = a.client.Playgrounds.Delete(ctx(), id) })
}

func attach(ids ...int64) *[]fibe.EnvPackAttachmentInput {
	rows := make([]fibe.EnvPackAttachmentInput, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, fibe.EnvPackAttachmentInput{EnvPackID: id})
	}
	return &rows
}

func attachNamed(id int64, names ...string) fibe.EnvPackAttachmentInput {
	return fibe.EnvPackAttachmentInput{EnvPackID: id, ServiceNames: &names}
}

func emptyAttachments() *[]fibe.EnvPackAttachmentInput {
	rows := []fibe.EnvPackAttachmentInput{}
	return &rows
}

// ---------------------------------------------------------------- assertions

func requireEnvPackError(t *testing.T, surface string, err error, code string, status int) *fibe.APIError {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected %s (%d), got success", surface, code, status)
	}
	var apiErr *fibe.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("%s: expected *fibe.APIError %s, got %T: %v", surface, code, err, err)
	}
	if apiErr.Code != code || apiErr.StatusCode != status {
		t.Fatalf("%s: expected %s (%d), got %s (%d): %s", surface, code, status, apiErr.Code, apiErr.StatusCode, apiErr.Message)
	}
	return apiErr
}

func requireNoLeak(t *testing.T, what string, value any, secret string) {
	t.Helper()
	raw, err := json.Marshal(value)
	requireNoError(t, err)
	if strings.Contains(string(raw), secret) {
		t.Fatalf("%s leaked a pack value: %s", what, raw)
	}
}

func attachmentPackIDs(target *fibe.EnvPackTarget) []int64 {
	ids := make([]int64, 0, len(target.EnvPackAttachments))
	for _, row := range target.EnvPackAttachments {
		ids = append(ids, row.EnvPackID)
	}
	return ids
}

func requireAttachmentOrder(t *testing.T, surface string, target *fibe.EnvPackTarget, want ...int64) {
	t.Helper()
	if target == nil {
		t.Fatalf("%s: nil attachment target", surface)
	}
	got := attachmentPackIDs(target)
	if len(got) != len(want) {
		t.Fatalf("%s: attachments %v, want %v", surface, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: attachments %v, want %v", surface, got, want)
		}
		if target.EnvPackAttachments[i].Position != i {
			t.Fatalf("%s: attachment %d has position %d, want %d", surface, got[i], target.EnvPackAttachments[i].Position, i)
		}
	}
}

func attachmentFor(t *testing.T, target *fibe.EnvPackTarget, packID int64) fibe.EnvPackAttachment {
	t.Helper()
	for _, row := range target.EnvPackAttachments {
		if row.EnvPackID == packID {
			return row
		}
	}
	t.Fatalf("pack %d is not attached: %v", packID, attachmentPackIDs(target))
	return fibe.EnvPackAttachment{}
}

// requireNames treats nil and an empty list as "all services".
func requireNames(t *testing.T, surface string, row fibe.EnvPackAttachment, want ...string) {
	t.Helper()
	if len(row.ServiceNames) != len(want) {
		t.Fatalf("%s: pack %d service_names %v, want %v", surface, row.EnvPackID, row.ServiceNames, want)
	}
	for i := range want {
		if row.ServiceNames[i] != want[i] {
			t.Fatalf("%s: pack %d service_names %v, want %v", surface, row.EnvPackID, row.ServiceNames, want)
		}
	}
}

func requireSameGrant(t *testing.T, surface string, before, after fibe.EnvPackAttachment) {
	t.Helper()
	if before.GrantingSubject != after.GrantingSubject || before.GrantedAt == nil || after.GrantedAt == nil || !before.GrantedAt.Equal(*after.GrantedAt) {
		t.Fatalf("%s: grant for pack %d changed: %s@%v -> %s@%v", surface, before.EnvPackID,
			before.GrantingSubject, before.GrantedAt, after.GrantingSubject, after.GrantedAt)
	}
}

// ---------------------------------------------------------------- raw HTTP

// envPackRaw sends a request without any client-side validation so the server's
// own contract can be checked. It returns the status and decoded JSON object.
func envPackRaw(t *testing.T, a *envPackActor, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		requireNoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx(), method, strings.TrimRight(a.client.BaseURL(), "/")+path, reader)
	requireNoError(t, err)
	req.Header.Set("Authorization", "Bearer "+a.key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: integrationHTTPTimeout()}).Do(req)
	requireNoError(t, err, "raw request "+method+" "+path)
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	requireNoError(t, err)
	out := map[string]any{}
	if len(bytes.TrimSpace(data)) > 0 {
		_ = json.Unmarshal(data, &out)
	}
	return res.StatusCode, out
}

func rawErrorCode(body map[string]any) string {
	if e, ok := body["error"].(map[string]any); ok {
		code, _ := e["code"].(string)
		return code
	}
	return ""
}

// ---------------------------------------------------------------- surfaces

// envPackSurface is one way to drive the same ENV-pack operations. Errors are
// normalised to *fibe.APIError so every surface asserts the same stable codes.
type envPackSurface interface {
	Name() string
	CreatePack(t *testing.T, name string, env map[string]string) (*fibe.EnvPack, error)
	GetPack(t *testing.T, id int64) (*fibe.EnvPack, error)
	ListPacks(t *testing.T) ([]fibe.EnvPack, error)
	UpdatePack(t *testing.T, id int64, p *fibe.EnvPackUpdateParams) (*fibe.EnvPack, error)
	DeletePack(t *testing.T, id int64) error
	Attachments(t *testing.T, target string, id int64) (*fibe.EnvPackTarget, error)
	// Replace with rows == nil omits env_pack_attachments (preserve); a pointer to an empty slice sends [].
	Replace(t *testing.T, target string, id int64, rows *[]fibe.EnvPackAttachmentInput) (*fibe.EnvPackTarget, error)
	Reorder(t *testing.T, target string, id int64, order []int64) (*fibe.EnvPackTarget, error)
	Detach(t *testing.T, target string, id, packID int64) (*fibe.EnvPackTarget, error)
	// Retarget with names == nil selects all services.
	Retarget(t *testing.T, target string, id, packID int64, names *[]string) (*fibe.EnvPackTarget, error)
	Renew(t *testing.T, target string, id, packID int64) (*fibe.EnvPackTarget, error)
	RerunPlayground(t *testing.T, id int64, name string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error)
	RerunTask(t *testing.T, identifier string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error)
	LaunchCompose(t *testing.T, name string, hostID int64, compose string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.LaunchResult, error)
	LaunchSpec(t *testing.T, spec, name string, hostID int64, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error)
}

func surfacesFor(t *testing.T, a *envPackActor) []envPackSurface {
	t.Helper()
	return []envPackSurface{sdkSurface{c: a.client}, cliSurface{key: a.key, domain: a.domain}, newMCPSurface(t, a)}
}

func attachmentID(t *testing.T, s envPackSurface, target string, id, packID int64) int64 {
	t.Helper()
	current, err := s.Attachments(t, target, id)
	requireNoError(t, err, s.Name()+" read attachments before renew")
	return attachmentFor(t, current, packID).ID
}

// --- Go SDK

type sdkSurface struct{ c *fibe.Client }

func (s sdkSurface) Name() string { return "sdk" }
func (s sdkSurface) CreatePack(t *testing.T, name string, env map[string]string) (*fibe.EnvPack, error) {
	return s.c.EnvPacks.Create(ctx(), &fibe.EnvPackCreateParams{Name: name, Env: env})
}
func (s sdkSurface) GetPack(t *testing.T, id int64) (*fibe.EnvPack, error) {
	return s.c.EnvPacks.Get(ctx(), id)
}
func (s sdkSurface) ListPacks(t *testing.T) ([]fibe.EnvPack, error) {
	out, err := s.c.EnvPacks.List(ctx(), &fibe.EnvPackListParams{PerPage: 100})
	if err != nil {
		return nil, err
	}
	return out.Data, nil
}
func (s sdkSurface) UpdatePack(t *testing.T, id int64, p *fibe.EnvPackUpdateParams) (*fibe.EnvPack, error) {
	return s.c.EnvPacks.Update(ctx(), id, p)
}
func (s sdkSurface) DeletePack(t *testing.T, id int64) error { return s.c.EnvPacks.Delete(ctx(), id) }
func (s sdkSurface) Attachments(t *testing.T, target string, id int64) (*fibe.EnvPackTarget, error) {
	return s.c.EnvPacks.Attachments(ctx(), target, id)
}
func (s sdkSurface) Replace(t *testing.T, target string, id int64, rows *[]fibe.EnvPackAttachmentInput) (*fibe.EnvPackTarget, error) {
	return s.c.EnvPacks.ReplaceAttachments(ctx(), target, id, &fibe.EnvPackAttachmentsParams{Attachments: rows})
}
func (s sdkSurface) Reorder(t *testing.T, target string, id int64, order []int64) (*fibe.EnvPackTarget, error) {
	return s.c.EnvPacks.ReorderAttachments(ctx(), target, id, order)
}
func (s sdkSurface) Detach(t *testing.T, target string, id, packID int64) (*fibe.EnvPackTarget, error) {
	return s.c.EnvPacks.DetachAttachment(ctx(), target, id, packID)
}
func (s sdkSurface) Retarget(t *testing.T, target string, id, packID int64, names *[]string) (*fibe.EnvPackTarget, error) {
	return s.c.EnvPacks.RetargetAttachment(ctx(), target, id, packID, names)
}
func (s sdkSurface) Renew(t *testing.T, target string, id, packID int64) (*fibe.EnvPackTarget, error) {
	return s.c.EnvPacks.RenewGrant(ctx(), target, id, attachmentID(t, s, target, id, packID))
}
func (s sdkSurface) RerunPlayground(t *testing.T, id int64, name string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	p := &fibe.PlaygroundRerunParams{EnvPackAttachments: rows}
	if name != "" {
		p.Name = &name
	}
	return s.c.Playgrounds.Rerun(ctx(), id, p)
}
func (s sdkSurface) RerunTask(t *testing.T, identifier string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	return s.c.Tasks.RerunWithParamsByIdentifier(ctx(), identifier, &fibe.PlaygroundRerunParams{EnvPackAttachments: rows})
}
func (s sdkSurface) LaunchCompose(t *testing.T, name string, hostID int64, compose string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.LaunchResult, error) {
	return s.c.Launch.Create(ctx(), &fibe.LaunchParams{Name: name, ComposeYAML: compose, HostID: &hostID, EnvPackAttachments: rows})
}
func (s sdkSurface) LaunchSpec(t *testing.T, spec, name string, hostID int64, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	return s.c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{Name: name, SpecIdentifier: spec, HostID: &hostID, EnvPackAttachments: rows})
}

// --- built CLI binary

type cliSurface struct{ key, domain string }

func (s cliSurface) Name() string { return "cli" }

func (s cliSurface) run(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	initCLIBin(t)
	home := t.TempDir() // never read an ambient profile
	full := append([]string{"--api-key", s.key, "--domain", s.domain, "--output", "json"}, args...)
	cmd := exec.Command(cliBinPath, full...)
	cmd.Env = append(os.Environ(), "FIBE_OUTPUT=json", "FIBE_API_KEY="+s.key, "FIBE_DOMAIN="+s.domain, "HOME="+home, "XDG_CONFIG_HOME="+home)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, cliAPIError(stderr.String(), err)
	}
	return stdout.Bytes(), nil
}

func cliAPIError(stderr string, runErr error) error {
	start := strings.Index(stderr, "{")
	if start >= 0 {
		var payload struct {
			Error struct {
				Message string         `json:"message"`
				Code    string         `json:"code"`
				Status  int            `json:"status"`
				Details map[string]any `json:"details"`
			} `json:"error"`
		}
		if json.NewDecoder(strings.NewReader(stderr[start:])).Decode(&payload) == nil && payload.Error.Message != "" {
			return &fibe.APIError{StatusCode: payload.Error.Status, Code: payload.Error.Code, Message: payload.Error.Message, Details: payload.Error.Details}
		}
	}
	return fmt.Errorf("%w: %s", runErr, strings.TrimSpace(stderr))
}

func decodeOutput(t *testing.T, surface string, data []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("%s: decode %T from %q: %v", surface, out, data, err)
	}
}

func writeJSONFile(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	requireNoError(t, err)
	path := filepath.Join(t.TempDir(), "payload.json")
	requireNoError(t, os.WriteFile(path, raw, 0o600))
	return path
}

// selectionFile is the --from-file body: nil rows omit the field.
func selectionFile(t *testing.T, rows *[]fibe.EnvPackAttachmentInput) string {
	return writeJSONFile(t, fibe.EnvPackAttachmentsParams{Attachments: rows})
}

func (s cliSurface) pack(t *testing.T, args ...string) (*fibe.EnvPack, error) {
	t.Helper()
	out, err := s.run(t, args...)
	if err != nil {
		return nil, err
	}
	var pack fibe.EnvPack
	decodeOutput(t, s.Name(), out, &pack)
	return &pack, nil
}

func (s cliSurface) target(t *testing.T, args ...string) (*fibe.EnvPackTarget, error) {
	t.Helper()
	out, err := s.run(t, args...)
	if err != nil {
		return nil, err
	}
	var target fibe.EnvPackTarget
	decodeOutput(t, s.Name(), out, &target)
	return &target, nil
}

func (s cliSurface) CreatePack(t *testing.T, name string, env map[string]string) (*fibe.EnvPack, error) {
	return s.pack(t, "env-packs", "create", "--from-file", writeJSONFile(t, map[string]any{"name": name, "env": env}))
}
func (s cliSurface) GetPack(t *testing.T, id int64) (*fibe.EnvPack, error) {
	return s.pack(t, "env-packs", "get", strconv.FormatInt(id, 10))
}
func (s cliSurface) ListPacks(t *testing.T) ([]fibe.EnvPack, error) {
	out, err := s.run(t, "env-packs", "list", "--per-page", "100")
	if err != nil {
		return nil, err
	}
	var list struct{ Data []fibe.EnvPack }
	decodeOutput(t, s.Name(), out, &list)
	return list.Data, nil
}
func (s cliSurface) UpdatePack(t *testing.T, id int64, p *fibe.EnvPackUpdateParams) (*fibe.EnvPack, error) {
	args := []string{"env-packs", "update", strconv.FormatInt(id, 10)}
	if p.Env != nil {
		args = append(args, "--from-file", writeJSONFile(t, map[string]any{"env": *p.Env}))
	}
	if p.Name != nil {
		args = append(args, "--name", *p.Name)
	}
	if p.Version != nil {
		args = append(args, "--version", strconv.FormatInt(*p.Version, 10))
	}
	return s.pack(t, args...)
}
func (s cliSurface) DeletePack(t *testing.T, id int64) error {
	_, err := s.run(t, "env-packs", "delete", strconv.FormatInt(id, 10), "--yes")
	return err
}
func (s cliSurface) Attachments(t *testing.T, target string, id int64) (*fibe.EnvPackTarget, error) {
	return s.target(t, "env-packs", "attachments", "get", target, strconv.FormatInt(id, 10))
}
func (s cliSurface) Replace(t *testing.T, target string, id int64, rows *[]fibe.EnvPackAttachmentInput) (*fibe.EnvPackTarget, error) {
	return s.target(t, "env-packs", "attachments", "set", target, strconv.FormatInt(id, 10), "--from-file", selectionFile(t, rows))
}
func (s cliSurface) Reorder(t *testing.T, target string, id int64, order []int64) (*fibe.EnvPackTarget, error) {
	parts := make([]string, len(order))
	for i, pack := range order {
		parts[i] = strconv.FormatInt(pack, 10)
	}
	return s.target(t, "env-packs", "attachments", "reorder", target, strconv.FormatInt(id, 10), "--order", strings.Join(parts, ","))
}
func (s cliSurface) Detach(t *testing.T, target string, id, packID int64) (*fibe.EnvPackTarget, error) {
	return s.target(t, "env-packs", "attachments", "detach", target, strconv.FormatInt(id, 10), strconv.FormatInt(packID, 10))
}
func (s cliSurface) Retarget(t *testing.T, target string, id, packID int64, names *[]string) (*fibe.EnvPackTarget, error) {
	args := []string{"env-packs", "attachments", "retarget", target, strconv.FormatInt(id, 10), strconv.FormatInt(packID, 10)}
	if names == nil {
		args = append(args, "--all-services")
	} else {
		args = append(args, "--services", strings.Join(*names, ","))
	}
	return s.target(t, args...)
}
func (s cliSurface) Renew(t *testing.T, target string, id, packID int64) (*fibe.EnvPackTarget, error) {
	return s.target(t, "env-packs", "attachments", "renew", target, strconv.FormatInt(id, 10), strconv.FormatInt(packID, 10))
}
func (s cliSurface) playground(t *testing.T, args ...string) (*fibe.Playground, error) {
	t.Helper()
	out, err := s.run(t, args...)
	if err != nil {
		return nil, err
	}
	var pg fibe.Playground
	decodeOutput(t, s.Name(), out, &pg)
	return &pg, nil
}
func (s cliSurface) RerunPlayground(t *testing.T, id int64, name string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	args := []string{"playgrounds", "rerun", strconv.FormatInt(id, 10)}
	if name != "" {
		args = append(args, "--name", name)
	}
	if rows != nil {
		args = append(args, "--from-file", selectionFile(t, rows))
	}
	return s.playground(t, args...)
}
func (s cliSurface) RerunTask(t *testing.T, identifier string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	args := []string{"tasks", "rerun", identifier}
	if rows != nil {
		args = append(args, "--from-file", selectionFile(t, rows))
	}
	return s.playground(t, args...)
}
func launchSelectionFlag(t *testing.T, rows *[]fibe.EnvPackAttachmentInput) []string {
	if rows == nil {
		return nil
	}
	raw, err := json.Marshal(*rows)
	requireNoError(t, err)
	return []string{"--env-pack-attachments", string(raw)}
}
func (s cliSurface) LaunchCompose(t *testing.T, name string, hostID int64, compose string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.LaunchResult, error) {
	args := append([]string{"launch", "--compose", compose, "--name", name, "--host", strconv.FormatInt(hostID, 10)}, launchSelectionFlag(t, rows)...)
	out, err := s.run(t, args...)
	if err != nil {
		return nil, err
	}
	var result fibe.LaunchResult
	decodeOutput(t, s.Name(), out, &result)
	return &result, nil
}
func (s cliSurface) LaunchSpec(t *testing.T, spec, name string, hostID int64, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	args := append([]string{"launch", "--spec", spec, "--name", name, "--host", strconv.FormatInt(hostID, 10)}, launchSelectionFlag(t, rows)...)
	return s.playground(t, args...)
}

// --- MCP stdio server

type mcpSurface struct{ c *client.Client }

func (m mcpSurface) Name() string { return "mcp" }

func newMCPSurface(t *testing.T, a *envPackActor) mcpSurface {
	t.Helper()
	initCLIBin(t)
	home := t.TempDir()
	env := append(os.Environ(), "FIBE_API_KEY="+a.key, "FIBE_DOMAIN="+a.domain, "HOME="+home, "XDG_CONFIG_HOME="+home)
	c, err := client.NewStdioMCPClient(cliBinPath, env, "--api-key", a.key, "--domain", a.domain, "mcp", "serve", "--tools", "full")
	requireNoError(t, err, "start fibe mcp serve")
	t.Cleanup(func() { _ = c.Close() })
	initCtx, cancel := context.WithTimeout(ctx(), 30*time.Second)
	defer cancel()
	var init mcp.InitializeRequest
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcp.Implementation{Name: "fibe-sdk-env-packs-integration", Version: "1.0.0"}
	_, err = c.Initialize(initCtx, init)
	requireNoError(t, err, "initialize fibe mcp serve")
	return mcpSurface{c: c}
}

// call invokes one tool. A tool-level failure is returned as *fibe.APIError.
func (m mcpSurface) call(t *testing.T, tool string, args map[string]any, out any) error {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = tool
	req.Params.Arguments = args
	callCtx, cancel := context.WithTimeout(ctx(), 3*time.Minute)
	defer cancel()
	res, err := m.c.CallTool(callCtx, req)
	requireNoError(t, err, "mcp transport for "+tool)
	var text strings.Builder
	for _, content := range res.Content {
		text.WriteString(mcp.GetTextFromContent(content))
	}
	if res.IsError {
		var payload struct {
			Message string         `json:"message"`
			Code    string         `json:"code"`
			Status  int            `json:"status"`
			Details map[string]any `json:"details"`
		}
		if json.Unmarshal([]byte(text.String()), &payload) == nil && payload.Code != "" {
			return &fibe.APIError{StatusCode: payload.Status, Code: payload.Code, Message: payload.Message, Details: payload.Details}
		}
		return fmt.Errorf("mcp %s: %s", tool, text.String())
	}
	if out != nil {
		decodeOutput(t, m.Name(), []byte(text.String()), out)
	}
	return nil
}

func (m mcpSurface) mutate(t *testing.T, resource, operation string, payload map[string]any, out any) error {
	return m.call(t, "fibe_resource_mutate", map[string]any{"resource": resource, "operation": operation, "payload": payload}, out)
}

func (m mcpSurface) attachmentMutation(t *testing.T, operation, target string, id int64, extra map[string]any) (*fibe.EnvPackTarget, error) {
	t.Helper()
	payload := map[string]any{"target": target, "target_id": id}
	for k, v := range extra {
		payload[k] = v
	}
	var out fibe.EnvPackTarget
	if err := m.mutate(t, "env_pack", operation, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (m mcpSurface) CreatePack(t *testing.T, name string, env map[string]string) (*fibe.EnvPack, error) {
	var out fibe.EnvPack
	if err := m.mutate(t, "env_pack", "create", map[string]any{"name": name, "env": env}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (m mcpSurface) GetPack(t *testing.T, id int64) (*fibe.EnvPack, error) {
	var out fibe.EnvPack
	if err := m.call(t, "fibe_resource_get", map[string]any{"resource": "env_pack", "id": id}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (m mcpSurface) ListPacks(t *testing.T) ([]fibe.EnvPack, error) {
	var list struct{ Data []fibe.EnvPack }
	if err := m.call(t, "fibe_resource_list", map[string]any{"resource": "env_pack", "params": map[string]any{"per_page": 100}}, &list); err != nil {
		return nil, err
	}
	return list.Data, nil
}
func (m mcpSurface) UpdatePack(t *testing.T, id int64, p *fibe.EnvPackUpdateParams) (*fibe.EnvPack, error) {
	payload := map[string]any{"env_pack_id": id}
	if p.Name != nil {
		payload["name"] = *p.Name
	}
	if p.Env != nil {
		payload["env"] = *p.Env
	}
	if p.Version != nil {
		payload["version"] = *p.Version
	}
	var out fibe.EnvPack
	if err := m.mutate(t, "env_pack", "update", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (m mcpSurface) DeletePack(t *testing.T, id int64) error {
	return m.call(t, "fibe_resource_delete", map[string]any{"resource": "env_pack", "id": id, "confirm": true}, nil)
}
func (m mcpSurface) Attachments(t *testing.T, target string, id int64) (*fibe.EnvPackTarget, error) {
	var out fibe.EnvPackTarget
	if err := m.call(t, "fibe_env_pack_attachments_get", map[string]any{"target": target, "target_id": id}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (m mcpSurface) Replace(t *testing.T, target string, id int64, rows *[]fibe.EnvPackAttachmentInput) (*fibe.EnvPackTarget, error) {
	extra := map[string]any{}
	if rows != nil {
		extra["env_pack_attachments"] = *rows
	}
	return m.attachmentMutation(t, "attachments_replace", target, id, extra)
}
func (m mcpSurface) Reorder(t *testing.T, target string, id int64, order []int64) (*fibe.EnvPackTarget, error) {
	return m.attachmentMutation(t, "attachments_reorder", target, id, map[string]any{"order": order})
}
func (m mcpSurface) Detach(t *testing.T, target string, id, packID int64) (*fibe.EnvPackTarget, error) {
	return m.attachmentMutation(t, "attachment_detach", target, id, map[string]any{"env_pack_id": packID})
}
func (m mcpSurface) Retarget(t *testing.T, target string, id, packID int64, names *[]string) (*fibe.EnvPackTarget, error) {
	extra := map[string]any{"env_pack_id": packID}
	if names != nil {
		extra["service_names"] = *names
	}
	return m.attachmentMutation(t, "attachment_retarget", target, id, extra)
}
func (m mcpSurface) Renew(t *testing.T, target string, id, packID int64) (*fibe.EnvPackTarget, error) {
	return m.attachmentMutation(t, "grant_renew", target, id, map[string]any{"attachment_id": attachmentID(t, m, target, id, packID)})
}
func (m mcpSurface) rerun(t *testing.T, resource string, payload map[string]any, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	if rows != nil {
		payload["env_pack_attachments"] = *rows
	}
	var out fibe.Playground
	if err := m.mutate(t, resource, "rerun", payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (m mcpSurface) RerunPlayground(t *testing.T, id int64, name string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	payload := map[string]any{"playground_id": id}
	if name != "" {
		payload["name"] = name
	}
	return m.rerun(t, "playground", payload, rows)
}
func (m mcpSurface) RerunTask(t *testing.T, identifier string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	return m.rerun(t, "task", map[string]any{"id_or_name": identifier}, rows)
}
func (m mcpSurface) LaunchCompose(t *testing.T, name string, hostID int64, compose string, rows *[]fibe.EnvPackAttachmentInput) (*fibe.LaunchResult, error) {
	args := map[string]any{"name": name, "compose_yaml": compose, "host_id_or_name": strconv.FormatInt(hostID, 10)}
	if rows != nil {
		args["env_pack_attachments"] = *rows
	}
	var out fibe.LaunchResult
	if err := m.call(t, "fibe_launch", args, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (m mcpSurface) LaunchSpec(t *testing.T, spec, name string, hostID int64, rows *[]fibe.EnvPackAttachmentInput) (*fibe.Playground, error) {
	args := map[string]any{"name": name, "spec_id_or_name": spec, "host_id_or_name": strconv.FormatInt(hostID, 10)}
	if rows != nil {
		args["env_pack_attachments"] = *rows
	}
	var out fibe.Playground
	if err := m.call(t, "fibe_launch", args, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
