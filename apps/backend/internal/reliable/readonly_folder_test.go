package reliable

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadOnlyFolderInspectionAllowsToolsCollectionWithoutGit(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	project := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(project, "codex-home"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(project, "codex-home", "config.toml"), []byte("native fixture"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("source"), 0600))
	require.Error(t, localProjectBoundary(project, e.Root))
	task := &Task{ID: "readonly-tools", Project: project, Goal: "Read project source", ReadOnly: true, State: stateQueued}
	require.NoError(t, e.Store.Create(task))
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	saved, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateInspected, saved.State)
	require.Empty(t, saved.Worktree)
	_, err = os.Stat(filepath.Join(project, ".git"))
	require.True(t, os.IsNotExist(err))
}

func TestHTTPReadOnlyToolsCollectionDoesNotCreateGitOrWorktree(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(e.wg.Wait)
	project := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(project, "codex-home"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(project, "codex-home", "config.toml"), []byte("fixture"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("original"), 0600))
	h := &HTTP{Engine: e, Token: "readonly-folder-test"}
	raw, err := json.Marshal(createRequest{Project: project, Goal: "Read source", ReadOnly: true})
	require.NoError(t, err)
	response := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks", string(raw), h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	var task Task
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &task))
	require.Eventually(t, func() bool {
		saved, getErr := e.Store.Get(task.ID)
		return getErr == nil && saved.State == stateInspected
	}, 5*time.Second, 10*time.Millisecond)
	saved, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Empty(t, saved.Worktree)
	_, err = os.Stat(filepath.Join(project, ".git"))
	require.True(t, os.IsNotExist(err))
	require.Error(t, selectedProjectBoundary(project, e.Root, false))
}
