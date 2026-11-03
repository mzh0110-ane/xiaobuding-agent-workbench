package reliable

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resultPreviewFixture(t *testing.T) (*Engine, *Task, *HTTP) {
	t.Helper()
	e, task := preparedCancelledTask(t, &fixtureAgent{}, 1)
	for name, body := range map[string]string{
		"index.html": "<!doctype html><title>Candidate preview</title><p>Current candidate</p>",
		"app.txt":    "candidate source\n", ".env": "SECRET_FIXTURE=do-not-serve\n",
	} {
		if err := os.WriteFile(filepath.Join(task.Worktree, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return e, task, &HTTP{Engine: e, Token: "preview-action"}
}

func TestResultPreviewStartsOnlyAfterExplicitActionAndCloses(t *testing.T) {
	e, task, handler := resultPreviewFixture(t)
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/preview"
	status := httpRequest(handler, http.MethodGet, endpoint, "", "")
	if status.Code != http.StatusOK {
		t.Fatalf("preview discovery must be available: %d %s", status.Code, status.Body)
	}
	var report map[string]any
	if json.Unmarshal(status.Body.Bytes(), &report) != nil || report["running"] != false || report["supported"] != true {
		t.Fatalf("discovery must not start a server: %s", status.Body)
	}
	denied := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start"}`, "")
	if denied.Code != http.StatusForbidden {
		t.Fatal("preview start did not require explicit browser action")
	}
	started := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start"}`, handler.Token)
	if started.Code != http.StatusOK || json.Unmarshal(started.Body.Bytes(), &report) != nil {
		t.Fatalf("preview start failed: %d %s", started.Code, started.Body)
	}
	url, _ := report["url"].(string)
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatalf("preview must use a separate loopback origin: %s", url)
	}
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if readErr != nil || !strings.Contains(string(body), "Current candidate") {
		t.Fatalf("preview did not read the owned candidate: %s %v", body, readErr)
	}
	if response, err = http.Get(url + ".env"); err == nil {
		_ = response.Body.Close()
		if response.StatusCode == http.StatusOK {
			t.Fatal("preview exposed an environment file")
		}
	}
	if err = e.Close(); err != nil {
		t.Fatal(err)
	}
	if response, err = http.Get(url); err == nil {
		_ = response.Body.Close()
		t.Fatal("backend close left its preview server running")
	}
}

func TestResultFilesExposeOnlyChangedSafeText(t *testing.T) {
	_, task, handler := resultPreviewFixture(t)
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/files"
	listed := httpRequest(handler, http.MethodGet, endpoint, "", "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"path":"app.txt"`) || strings.Contains(listed.Body.String(), `"path":".env"`) {
		t.Fatalf("changed source listing missing or unsafe: %d %s", listed.Code, listed.Body)
	}
	read := httpRequest(handler, http.MethodGet, endpoint+"?path=app.txt", "", "")
	if read.Code != http.StatusOK || !strings.Contains(read.Body.String(), "candidate source") {
		t.Fatalf("changed source not readable: %d %s", read.Code, read.Body)
	}
	for _, name := range []string{".env", "../state.db", ".git", "AGENTS.md", "node_modules/package.json"} {
		bad := httpRequest(handler, http.MethodGet, endpoint+"?path="+name, "", "")
		if bad.Code == http.StatusOK {
			t.Fatalf("unsafe result source was readable: %s", name)
		}
	}
}

func TestResultPreviewRejectsTaskExecutionAndForeignWorktree(t *testing.T) {
	e, task, handler := resultPreviewFixture(t)
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/preview"
	if _, err := e.Store.Update(task.ID, func(saved *Task) error { saved.State = stateImplementing; return nil }); err != nil {
		t.Fatal(err)
	}
	busy := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start"}`, handler.Token)
	if busy.Code == http.StatusOK {
		t.Fatal("preview started while candidate was being modified")
	}
	if _, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.State, saved.Worktree = stateCancelled, saved.Project
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	foreign := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start"}`, handler.Token)
	if foreign.Code == http.StatusOK {
		t.Fatal("preview started against the original instead of the owned candidate")
	}
}

func TestResultPreviewReportsSourceChangeAndStillStops(t *testing.T) {
	_, task, handler := resultPreviewFixture(t)
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/preview"
	started := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start"}`, handler.Token)
	if started.Code != http.StatusOK {
		t.Fatalf("preview positive control failed: %s", started.Body)
	}
	if err := os.WriteFile(filepath.Join(task.Worktree, "app.txt"), []byte("manual later edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	status := httpRequest(handler, http.MethodGet, endpoint, "", "")
	var report map[string]any
	if status.Code != http.StatusOK || json.Unmarshal(status.Body.Bytes(), &report) != nil || report["stale"] != true {
		t.Fatalf("a running preview must report changed source: %d %s", status.Code, status.Body)
	}
	stopped := httpRequest(handler, http.MethodPost, endpoint, `{"action":"stop"}`, handler.Token)
	if stopped.Code != http.StatusOK || json.Unmarshal(stopped.Body.Bytes(), &report) != nil || report["running"] != false {
		t.Fatalf("source changes must not prevent stopping preview: %d %s", stopped.Code, stopped.Body)
	}
}

func TestResultFilesBoundResponsesAndRejectRedirectedFiles(t *testing.T) {
	_, task, handler := resultPreviewFixture(t)
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/files?path="
	for name, content := range map[string][]byte{
		"large.txt":        []byte(strings.Repeat("x", (512<<10)+1)),
		"binary.txt":       {0xff, 0, 0xfe},
		"credentials.json": []byte(`{"fixture":"do-not-serve"}`),
	} {
		if err := os.WriteFile(filepath.Join(task.Worktree, name), content, 0600); err != nil {
			t.Fatal(err)
		}
		response := httpRequest(handler, http.MethodGet, endpoint+url.QueryEscape(name), "", "")
		if response.Code == http.StatusOK {
			t.Fatalf("unsafe or oversized text was exposed: %s", name)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("OUTSIDE_SENTINEL"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(task.Worktree, "redirect.txt")); err != nil {
		t.Logf("symlink creation unavailable, basic bounds already verified: %v", err)
		return
	}
	response := httpRequest(handler, http.MethodGet, endpoint+"redirect.txt", "", "")
	if response.Code == http.StatusOK || strings.Contains(response.Body.String(), "OUTSIDE_SENTINEL") {
		t.Fatal("result source followed a file link outside the candidate")
	}
}

func TestResultPreviewRejectsCrossOriginAssetsAndStopsBeforeExecution(t *testing.T) {
	e, task, handler := resultPreviewFixture(t)
	started := httpRequest(handler, http.MethodPost, "/api/v1/reliable/tasks/"+task.ID+"/preview", `{"action":"start"}`, handler.Token)
	var report map[string]any
	if started.Code != http.StatusOK || json.Unmarshal(started.Body.Bytes(), &report) != nil {
		t.Fatalf("preview positive control failed: %s", started.Body)
	}
	address, _ := report["url"].(string)
	request, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "https://unrelated.example")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatal("preview assets allowed an unrelated website origin")
	}
	_, release, err := e.reserve(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if response, err = http.Get(address); err == nil {
		_ = response.Body.Close()
		t.Fatal("Agent admission left a preview reading concurrently modified source")
	}
}

func TestResultPreviewOffersGeneratedHtmlArtifacts(t *testing.T) {
	_, task, handler := resultPreviewFixture(t)
	if err := os.Remove(filepath.Join(task.Worktree, "index.html")); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(task.Worktree, "artifacts", "physics", "result.html")
	if err := os.MkdirAll(filepath.Dir(artifact), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("<p>Generated interactive result</p>"), 0600); err != nil {
		t.Fatal(err)
	}
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/preview"
	status := httpRequest(handler, http.MethodGet, endpoint, "", "")
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), "artifacts/physics/result.html") {
		t.Fatalf("generated changed HTML must be discoverable: %d %s", status.Code, status.Body)
	}
	started := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start","entry":"artifacts/physics/result.html"}`, handler.Token)
	var report map[string]any
	if started.Code != http.StatusOK || json.Unmarshal(started.Body.Bytes(), &report) != nil {
		t.Fatalf("generated artifact preview did not start: %d %s", started.Code, started.Body)
	}
	address, _ := report["url"].(string)
	response, err := http.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || !strings.Contains(string(body), "Generated interactive result") {
		t.Fatalf("wrong generated artifact rendered: %s %v", body, err)
	}
}
