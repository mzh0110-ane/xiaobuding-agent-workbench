package reliable

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func archivedAt(t *testing.T, task *Task) string {
	t.Helper()
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err = json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	value, _ := fields["archived_at"].(string)
	return value
}

func TestDesktopArchivePersistsAndKeepsTheSealedCandidate(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "desktop-action"}
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	before, _ := e.Store.Get(task.ID)
	w := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/archive", `{}`, h.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("task archive is unavailable: %d %s", w.Code, w.Body)
	}
	got, _ := e.Store.Get(task.ID)
	if archivedAt(t, got) == "" || got.State != before.State || got.Round != before.Round || got.EvidenceHash != before.EvidenceHash {
		t.Fatalf("archive changed the execution contract: %+v", got)
	}
	if err := e.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("an archived candidate was applied without first restoring it")
	}
	if err := e.Store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(filepath.Join(e.Root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	e.Store = store
	t.Cleanup(func() { _ = store.Close() })
	got, _ = e.Store.Get(task.ID)
	if archivedAt(t, got) == "" {
		t.Fatal("archive was lost after reopening the database")
	}
	w = httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/unarchive", `{}`, h.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("restore failed: %d %s", w.Code, w.Body)
	}
	if err := e.Accept(context.Background(), task.ID); err != nil {
		t.Fatalf("restoring invalidated the original passing evidence: %v", err)
	}
}

func TestDesktopArchiveCannotRaceExecutionOrReleaseAnUnfinishedProject(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "desktop-action"}
	_, _ = e.Store.Update(task.ID, func(saved *Task) error { saved.State = stateBlocked; return nil })
	_, release, err := e.reserve(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	w := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/archive", `{}`, h.Token)
	if w.Code != http.StatusConflict {
		t.Fatalf("a reserved execution was archived: %d", w.Code)
	}
	release()
	w = httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/archive", `{}`, h.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("resting task was not archived: %d %s", w.Code, w.Body)
	}
	if err = e.Store.Create(&Task{ID: "duplicate", Project: task.Project, State: stateQueued}); err == nil {
		t.Fatal("archiving released the unfinished project's reservation")
	}
	if err = e.RunTask(context.Background(), task.ID); err == nil || !strings.Contains(err.Error(), "archived") {
		t.Fatalf("archived task started execution: %v", err)
	}
}

func TestDesktopFolderActionsReportUnavailableDirectories(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "desktop-action"}
	_, _ = e.Store.Update(task.ID, func(saved *Task) error { saved.Project = filepath.Join(t.TempDir(), "missing"); return nil })
	for _, action := range []string{"open-project", "open-evidence", "open"} {
		w := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/"+action, `{}`, h.Token)
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "folder_unavailable") {
			t.Fatalf("missing folder has no safe recovery feedback for %s: %d %s", action, w.Code, w.Body)
		}
	}
}

func TestDesktopFolderActionsUseOnlyTaskOwnedTargets(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	var opened []string
	h := &HTTP{Engine: e, Token: "desktop-action", OpenFolder: func(_ context.Context, path string) error { opened = append(opened, path); return nil }}
	worktree := filepath.Join(e.Root, "worktrees", task.ID)
	evidence := filepath.Join(e.Root, "evidence", task.ID)
	for _, path := range []string{worktree, evidence} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = e.Store.Update(task.ID, func(saved *Task) error { saved.Worktree = worktree; return nil })
	for action, expected := range map[string]string{"open-project": task.Project, "open": worktree, "open-evidence": evidence} {
		w := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/"+action, `{}`, h.Token)
		if w.Code != http.StatusOK || projectKey(opened[len(opened)-1]) != projectKey(expected) {
			t.Fatalf("wrong folder opened for %s: %d %v", action, w.Code, opened)
		}
	}
	before := len(opened)
	for _, request := range []struct{ method, path, body, token string }{
		{"POST", "/api/v1/reliable/tasks/task/open-project", `{}`, ""},
		{"GET", "/api/v1/reliable/tasks/task/open-project", "", ""},
		{"POST", "/api/v1/reliable/tasks/missing/open-project", `{}`, h.Token},
		{"POST", "/api/v1/reliable/tasks/task/open-project", `{"path":"arbitrary"}`, h.Token},
	} {
		w := httpRequest(h, request.method, request.path, request.body, request.token)
		if w.Code < 400 || len(opened) != before {
			t.Fatalf("an untrusted folder request invoked the desktop: %d", w.Code)
		}
	}
	_, _ = e.Store.Update(task.ID, func(saved *Task) error { saved.Worktree = task.Project; return nil })
	w := httpRequest(h, "POST", "/api/v1/reliable/tasks/task/open", `{}`, h.Token)
	if w.Code != http.StatusConflict || len(opened) != before {
		t.Fatal("an out-of-boundary working copy was opened")
	}
}

func TestDesktopArchiveEligibilityIdempotenceAndResumeGuard(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	e.Providers = testProviders()
	h := &HTTP{Engine: e, Token: "desktop-action"}
	for _, state := range []string{stateQueued, statePreparing, stateAuditing, stateImplementing, stateChecking, stateReviewing, stateApplying, stateCancelling} {
		_, _ = e.Store.Update(task.ID, func(saved *Task) error { saved.State = state; return nil })
		w := httpRequest(h, "POST", "/api/v1/reliable/tasks/task/archive", `{}`, h.Token)
		if w.Code != http.StatusConflict {
			t.Fatalf("active state %s was archived", state)
		}
	}
	_, _ = e.Store.Update(task.ID, func(saved *Task) error { saved.State = stateBlocked; return nil })
	for range 2 {
		w := httpRequest(h, "POST", "/api/v1/reliable/tasks/task/archive", `{}`, h.Token)
		if w.Code != http.StatusOK {
			t.Fatalf("idempotent archive failed: %s", w.Body)
		}
	}
	got, _ := e.Store.Get(task.ID)
	if archivedAt(t, got) == "" {
		t.Fatal("no archive metadata persisted")
	}
	w := httpRequest(h, "POST", "/api/v1/reliable/tasks/task/resume", `{}`, h.Token)
	if w.Code != http.StatusConflict {
		t.Fatalf("the HTTP resume entry did not reject an archived task: %d", w.Code)
	}
	if _, err := e.UpdateModelPlan(task.ID, got.PlanRevision, e.defaultPlan()); err == nil {
		t.Fatal("an archived task's assignments were edited")
	}
}
