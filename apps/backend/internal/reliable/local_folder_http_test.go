package reliable

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func localFolderHTTP(t *testing.T, runner AgentRunner) *HTTP {
	t.Helper()
	root := t.TempDir()
	store, err := OpenStore(filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	engine := &Engine{Root: root, Store: store, Runner: runner, Providers: testProviders()}
	t.Cleanup(func() { _ = engine.Close() })
	return &HTTP{Engine: engine, Token: "local-folder-capability"}
}

func createLocalFolderTask(t *testing.T, h *HTTP, project string, commands []string) *Task {
	t.Helper()
	raw, err := json.Marshal(createRequest{Project: project, Goal: "Update the project and preserve current behavior", Commands: commands})
	if err != nil {
		t.Fatal(err)
	}
	w := httpRequest(h, "POST", "/api/v1/reliable/tasks", string(raw), h.Token)
	if w.Code != http.StatusAccepted {
		t.Fatalf("a normal local folder must start without manual Git setup: %d %s", w.Code, w.Body)
	}
	var task Task
	if err = json.Unmarshal(w.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	return &task
}

func awaitLocalFolderTask(t *testing.T, h *HTTP, id string) *Task {
	t.Helper()
	deadline := time.NewTimer(60 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		task, err := h.Engine.Store.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if task.State == stateAwaitingAcceptance || task.State == stateBlocked || task.State == stateExhausted {
			h.Engine.mu.Lock()
			active := h.Engine.active[id] != nil
			h.Engine.mu.Unlock()
			if !active {
				return task
			}
		}
		select {
		case <-deadline.C:
			t.Fatal("local project preparation did not reach a bounded outcome")
		case <-tick.C:
		}
	}
}

func plainAppFolder(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "app.txt"), []byte("before\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return project
}

func TestPlainFolderIsStoredLocallyAndReusedAfterAcceptance(t *testing.T) {
	h := localFolderHTTP(t, &fixtureAgent{})
	project := plainAppFolder(t)
	task := createLocalFolderTask(t, h, project, []string{passingCheck})
	got := awaitLocalFolderTask(t, h, task.ID)
	if got.State != stateAwaitingAcceptance {
		t.Fatalf("local folder did not finish: %s %s", got.State, got.Reason)
	}
	first := strings.TrimSpace(gitTest(t, project, "rev-parse", "HEAD"))
	gitTest(t, project, "config", "core.autocrlf", "true")
	if content := gitTest(t, project, "show", "HEAD:app.txt"); content != "before\n" {
		t.Fatal("the initial recovery point does not preserve the original source")
	}
	if remotes := gitTest(t, project, "remote"); remotes != "" {
		t.Fatal("automatic local storage configured a remote")
	}
	if err := h.Engine.Accept(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "app.txt"), []byte("second draft\n"), 0600); err != nil {
		t.Fatal(err)
	}
	next := createLocalFolderTask(t, h, project, []string{passingCheck})
	got = awaitLocalFolderTask(t, h, next.ID)
	if got.State != stateAwaitingAcceptance {
		t.Fatalf("the next task still required manual Git work: %s %s", got.State, got.Reason)
	}
	if content := gitTest(t, project, "show", "HEAD:app.txt"); content != "second draft\n" {
		t.Fatal("the second local recovery point did not preserve the current draft")
	}
	if first == strings.TrimSpace(gitTest(t, project, "rev-parse", "HEAD")) {
		t.Fatal("the local recovery point was not advanced")
	}
}

func TestUnfinishedTaskIsRejectedBeforeAnotherLocalCheckpoint(t *testing.T) {
	h := localFolderHTTP(t, &fixtureAgent{})
	project := plainAppFolder(t)
	task := createLocalFolderTask(t, h, project, []string{passingCheck})
	if got := awaitLocalFolderTask(t, h, task.ID); got.State != stateAwaitingAcceptance {
		t.Fatalf("first candidate failed: %s", got.Reason)
	}
	head := gitTest(t, project, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(project, "app.txt"), []byte("new external draft\n"), 0600); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(createRequest{Project: project, Goal: "Another task", Commands: []string{passingCheck}})
	w := httpRequest(h, "POST", "/api/v1/reliable/tasks", string(raw), h.Token)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "unfinished task") {
		t.Fatalf("the existing task was not reserved before preparation: %d %s", w.Code, w.Body)
	}
	if gitTest(t, project, "rev-parse", "HEAD") != head {
		t.Fatal("a duplicate request changed the active task's original HEAD")
	}
}

func TestLocalStoragePreservesButDoesNotTrackSecretsOrDependencies(t *testing.T) {
	h := localFolderHTTP(t, &fixtureAgent{})
	project := plainAppFolder(t)
	private := []string{".env", "auth.json", "credentials.json", ".netrc", ".pypirc", ".npmrc", "id_ecdsa", "client.p12", "client.pfx", "node_modules/vendor/index.js"}
	for _, relative := range private {
		path := filepath.Join(project, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("private local fixture\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	task := createLocalFolderTask(t, h, project, []string{passingCheck})
	if got := awaitLocalFolderTask(t, h, task.ID); got.State != stateAwaitingAcceptance {
		t.Fatalf("safe local project failed: %s", got.Reason)
	}
	tracked := gitTest(t, project, "ls-files")
	for _, path := range private {
		if strings.Contains(tracked, path) {
			t.Fatalf("excluded local material was committed: %s", path)
		}
	}
	if raw, err := os.ReadFile(filepath.Join(project, ".env")); err != nil || string(raw) != "private local fixture\n" {
		t.Fatal("the local secret file was modified")
	}
}

func TestGitEnvironmentCannotRedirectLocalStorageOrCandidateIndex(t *testing.T) {
	h := localFolderHTTP(t, &fixtureAgent{})
	project := plainAppFolder(t)
	external := fixtureRepo(t)
	index := filepath.Join(external, ".git", "index")
	before, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_INDEX_FILE", index)
	task := createLocalFolderTask(t, h, project, []string{passingCheck})
	got := awaitLocalFolderTask(t, h, task.ID)
	if got.State != stateAwaitingAcceptance {
		t.Fatalf("inherited Git environment broke local preparation: %s", got.Reason)
	}
	after, err := os.ReadFile(index)
	if err != nil || string(after) != string(before) {
		t.Fatal("the controller changed an external Git index")
	}
}

func TestOverlappingLocalProjectsReserveOnePreparationSlot(t *testing.T) {
	h := localFolderHTTP(t, &fixtureAgent{})
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if err := h.Engine.Store.Create(&Task{ID: "reserved-parent", Project: parent, State: stateQueued, AutoSnapshot: true}); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(createRequest{Project: child, Goal: "Nested project", Commands: []string{passingCheck}})
	w := httpRequest(h, "POST", "/api/v1/reliable/tasks", string(raw), h.Token)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "unfinished task") {
		t.Fatalf("overlapping folder was not reserved before preparation: %d %s", w.Code, w.Body)
	}
	if _, err := os.Stat(filepath.Join(child, ".git")); !os.IsNotExist(err) {
		t.Fatal("the rejected overlapping project was initialized")
	}
}
