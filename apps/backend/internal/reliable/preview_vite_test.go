package reliable

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fakePreviewViteRuntime(t *testing.T, candidate string) {
	t.Helper()
	module := `export async function createServer(options) {
if (options.configFile !== false || options.server.hmr !== false || !options.server.fs.strict) throw Error('unsafe runtime configuration');
return {middlewares(request,response){response.setHeader('Content-Type','text/html');response.end('<p>VITE_CANDIDATE</p>')},async close(){}};
}`
	files := map[string]string{
		".gitignore":                           "node_modules/\n",
		"package.json":                         `{"scripts":{"dev":"vite","predev":"node should-never-run.cjs"}}`,
		"vite.config.mjs":                      `throw Error('project config must not execute in base preview');`,
		"node_modules/vite/package.json":       `{"type":"module"}`,
		"node_modules/vite/dist/node/index.js": module,
	}
	for name, contents := range files {
		full := filepath.Join(candidate, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResultPreviewRunsBareViteWithoutNpmScripts(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("owned dev-server process trees are attested on Windows")
	}
	_, task, handler := resultPreviewFixture(t)
	fakePreviewViteRuntime(t, task.Worktree)
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/preview"
	started := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start","entry":"vite"}`, handler.Token)
	var report map[string]any
	if started.Code != http.StatusOK || json.Unmarshal(started.Body.Bytes(), &report) != nil || report["kind"] != "vite" {
		t.Fatalf("bare Vite runtime was not started: %d %s", started.Code, started.Body)
	}
	url, _ := report["url"].(string)
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if readErr != nil || !strings.Contains(string(body), "VITE_CANDIDATE") {
		t.Fatalf("owned Vite middleware did not serve the candidate: %s %v", body, readErr)
	}
	stopped := httpRequest(handler, http.MethodPost, endpoint, `{"action":"stop"}`, handler.Token)
	if stopped.Code != http.StatusOK {
		t.Fatalf("owned Vite shutdown failed: %s", stopped.Body)
	}
	if response, err = http.Get(url); err == nil {
		_ = response.Body.Close()
		t.Fatal("stopped Vite preview listener was left running")
	}
}

func TestResultPreviewReportsMissingViteDependencies(t *testing.T) {
	_, task, handler := resultPreviewFixture(t)
	if err := os.WriteFile(filepath.Join(task.Worktree, "package.json"), []byte(`{"scripts":{"dev":"vite"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	status := httpRequest(handler, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/preview", "", "")
	var report map[string]any
	if status.Code != http.StatusOK || json.Unmarshal(status.Body.Bytes(), &report) != nil || report["reason"] != "preview_dependencies_missing" || report["supported"] != false {
		t.Fatalf("source HTML must not masquerade as a runnable Vite build: %d %s", status.Code, status.Body)
	}
}

func TestResultPreviewPrefersAvailableViteSourceOverOldBuild(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("owned Vite runtimes are available on Windows")
	}
	_, task, handler := resultPreviewFixture(t)
	fakePreviewViteRuntime(t, task.Worktree)
	if err := os.Mkdir(filepath.Join(task.Worktree, "dist"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(task.Worktree, "dist", "index.html"), []byte("old build"), 0600); err != nil {
		t.Fatal(err)
	}
	status := httpRequest(handler, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/preview", "", "")
	var report map[string]any
	if status.Code != http.StatusOK || json.Unmarshal(status.Body.Bytes(), &report) != nil || report["kind"] != "vite" {
		t.Fatalf("old build silently replaced available source preview: %d %s", status.Code, status.Body)
	}
}
