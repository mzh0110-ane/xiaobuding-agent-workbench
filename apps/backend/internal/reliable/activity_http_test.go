package reliable

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPActivityDoesNotInventOldHistory(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "activity-test"}
	response := httpRequest(h, "GET", "/api/v1/reliable/tasks/"+task.ID+"/activity", "", "")
	require.Equal(t, http.StatusOK, response.Code)
	var got struct {
		Events []any `json:"events"`
		Cursor int64 `json:"cursor"`
		Active bool  `json:"active"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &got))
	require.Empty(t, got.Events)
	require.Zero(t, got.Cursor)
	require.False(t, got.Active)
}

func TestHTTPReadOnlyInspectionDoesNotCreateGitOrWorktree(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(e.wg.Wait)
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("original\n"), 0600))
	h := &HTTP{Engine: e, Token: "inspect-test"}
	body, err := json.Marshal(map[string]any{"project": project, "goal": "了解当前项目", "read_only": true})
	require.NoError(t, err)
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks", string(body), h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	var task Task
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &task))
	require.Eventually(t, func() bool {
		current, getErr := e.Store.Get(task.ID)
		return getErr == nil && current.State == "inspected"
	}, 5*time.Second, 10*time.Millisecond)
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Zero(t, current.Round)
	require.Empty(t, current.Worktree)
	require.NotNil(t, current.Audit)
	_, err = os.Stat(filepath.Join(project, ".git"))
	require.True(t, os.IsNotExist(err), "inspection initialized Git")
	_, err = os.Stat(filepath.Join(e.Root, "worktrees", task.ID))
	require.True(t, os.IsNotExist(err), "inspection created another worktree")
	raw, err := os.ReadFile(filepath.Join(project, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, "original\n", string(raw))
}
