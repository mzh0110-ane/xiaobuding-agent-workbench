package reliable

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestResultPreviewRetainsStoppingOwnershipUntilCleanupCompletes(t *testing.T) {
	e, task, handler := resultPreviewFixture(t)
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/preview"
	started := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start"}`, handler.Token)
	if started.Code != http.StatusOK {
		t.Fatalf("preview positive control failed: %s", started.Body)
	}
	manager := e.previewManager()
	manager.mu.Lock()
	entry := manager.entries[task.ID]
	stopStarted, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	entry.stop = func() error {
		once.Do(func() { close(stopStarted) })
		<-release
		return errors.New("fixture cleanup unknown")
	}
	manager.mu.Unlock()
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	stopped := make(chan error, 1)
	go func() { stopped <- manager.StopTask(task.ID) }()
	<-stopStarted
	if err := os.WriteFile(filepath.Join(task.Worktree, "app.txt"), []byte("changed during cleanup\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stopping := httpRequest(handler, http.MethodGet, endpoint, "", "")
	var status map[string]any
	if json.Unmarshal(stopping.Body.Bytes(), &status) != nil || status["reason"] != "preview_stopping" || status["stale"] != true {
		t.Errorf("source change obscured authoritative stopping state: %s", stopping.Body)
	}
	concurrent := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start"}`, handler.Token)
	releaseOnce.Do(func() { close(release) })
	if err := <-stopped; err == nil {
		t.Fatal("the cleanup failure was hidden")
	}
	if concurrent.Code == http.StatusOK {
		t.Fatalf("new preview was admitted while prior cleanup was blocked: %s", concurrent.Body)
	}
	manager.mu.Lock()
	preserved := manager.entries[task.ID] == entry
	manager.mu.Unlock()
	if !preserved {
		t.Fatal("failed cleanup lost the authoritative old preview owner")
	}
	again := httpRequest(handler, http.MethodPost, endpoint, `{"action":"start"}`, handler.Token)
	if again.Code == http.StatusOK {
		t.Fatalf("cleanup-unknown preview was reported as running: %s", again.Body)
	}
	unknown := httpRequest(handler, http.MethodGet, endpoint, "", "")
	if json.Unmarshal(unknown.Body.Bytes(), &status) != nil || status["reason"] != "preview_cleanup_unknown" || status["stale"] != true {
		t.Fatalf("source change obscured cleanupunknown: %s", unknown.Body)
	}
}
