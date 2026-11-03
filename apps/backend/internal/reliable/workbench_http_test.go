package reliable

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func workbenchHTTPFixture(t *testing.T) (*HTTP, *Task) {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "state.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	task := &Task{ID: "long-project", Project: t.TempDir(), Goal: strings.Repeat("长期项目需求", 300), State: stateBlocked}
	task.Baseline = &Snapshot{Hash: "baseline-hash"}
	for i := range 60 {
		task.Messages = append(task.Messages, TaskMessage{ID: string(rune('A' + i)), Content: strings.Repeat("historical-message", 200), Status: "answered"})
	}
	task.PreviousContext = json.RawMessage(`{"private_context":"not-a-list-field"}`)
	require.NoError(t, store.Create(task))
	return &HTTP{Engine: &Engine{Store: store, Root: t.TempDir()}, Token: "workbench-test"}, task
}

func TestWorkbenchSummaryOmitsHistoryAndPreservesLegacyAPI(t *testing.T) {
	h, task := workbenchHTTPFixture(t)
	response := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks?view=summary", "", "")
	require.Equal(t, http.StatusOK, response.Code)
	require.Less(t, response.Body.Len(), 12000, "the list must not include message bodies")
	var result struct {
		Tasks []map[string]any `json:"tasks"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Len(t, result.Tasks, 1)
	require.Equal(t, task.ID, result.Tasks[0]["id"])
	require.NotEmpty(t, result.Tasks[0]["project_id"])
	for _, key := range []string{"messages", "baseline", "previous_context", "review", "audit"} {
		require.NotContains(t, result.Tasks[0], key)
	}
	legacy := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks", "", "")
	var old []*Task
	require.NoError(t, json.Unmarshal(legacy.Body.Bytes(), &old))
	require.Len(t, old[0].Messages, 60)
}

func TestWorkbenchConditionalReadAndMessagePagination(t *testing.T) {
	h, task := workbenchHTTPFixture(t)
	path := "/api/v1/reliable/tasks/" + task.ID + "?view=workspace"
	first := httpRequest(h, http.MethodGet, path, "", "")
	require.Equal(t, http.StatusOK, first.Code)
	require.NotEmpty(t, first.Header().Get("ETag"))
	var result struct {
		Task
		MessagesBefore int `json:"messages_before"`
		MessagesTotal  int `json:"messages_total"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &result))
	require.Len(t, result.Messages, 25)
	require.Equal(t, 35, result.MessagesBefore)
	require.Equal(t, 60, result.MessagesTotal)
	require.Equal(t, "baseline-hash", result.Baseline.Hash)
	require.Empty(t, result.PreviousContext)
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:38429"+path, nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("If-None-Match", first.Header().Get("ETag"))
	unchanged := httptest.NewRecorder()
	h.ServeHTTP(unchanged, request)
	require.Equal(t, http.StatusNotModified, unchanged.Code)
	require.Zero(t, unchanged.Body.Len())
	_, err := h.Engine.Store.Update(task.ID, func(saved *Task) error {
		saved.Messages = append(saved.Messages, TaskMessage{ID: "new", Content: "latest"})
		return nil
	})
	require.NoError(t, err)
	changed := httptest.NewRecorder()
	h.ServeHTTP(changed, request)
	require.Equal(t, http.StatusOK, changed.Code)
	require.NotEqual(t, first.Header().Get("ETag"), changed.Header().Get("ETag"))
	older := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/messages?before=35&limit=10", "", "")
	require.Equal(t, http.StatusOK, older.Code)
	var page struct {
		Messages []TaskMessage `json:"messages"`
		Before   int           `json:"before"`
	}
	require.NoError(t, json.Unmarshal(older.Body.Bytes(), &page))
	require.Len(t, page.Messages, 10)
	require.Equal(t, 25, page.Before)
	require.Equal(t, task.Messages[25].ID, page.Messages[0].ID)
}

func TestWorkbenchRejectsInvalidPaginationWithoutTaskMutation(t *testing.T) {
	h, task := workbenchHTTPFixture(t)
	for _, query := range []string{"before=-1", "limit=1000", "before=oops", "before=10000"} {
		response := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/messages?"+query, "", "")
		require.Equal(t, http.StatusBadRequest, response.Code, query)
	}
	current, err := h.Engine.Store.Get(task.ID)
	require.NoError(t, err)
	require.Len(t, current.Messages, 60)
	require.Equal(t, stateBlocked, current.State)
}

func TestWorkbenchSummarySearchFindsTextBeyondTheShortLabel(t *testing.T) {
	h, task := workbenchHTTPFixture(t)
	_, err := h.Engine.Store.Update(task.ID, func(saved *Task) error { saved.Goal += " needle-after-the-excerpt"; return nil })
	require.NoError(t, err)
	for _, query := range []struct {
		text  string
		count int
	}{{"needle-after-the-excerpt", 1}, {"absent-needle", 0}} {
		response := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks?view=summary&q="+query.text, "", "")
		var value struct {
			Tasks []map[string]any `json:"tasks"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &value))
		require.Len(t, value.Tasks, query.count)
		if query.count > 0 {
			require.Equal(t, true, value.Tasks[0]["search_match"])
		}
	}
}

func TestWorkbenchExecutionOwnershipIsVisibleUntilCleanupCompletes(t *testing.T) {
	h, task := workbenchHTTPFixture(t)
	h.Engine.active = map[string]context.CancelFunc{task.ID: func() {}}
	path := "/api/v1/reliable/tasks/" + task.ID + "?view=workspace"
	first := httpRequest(h, http.MethodGet, path, "", "")
	var running map[string]any
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &running))
	require.Equal(t, true, running["execution_active"])
	delete(h.Engine.active, task.ID)
	last := httpRequest(h, http.MethodGet, path, "", "")
	var stopped map[string]any
	require.NoError(t, json.Unmarshal(last.Body.Bytes(), &stopped))
	require.Equal(t, false, stopped["execution_active"])
	require.NotEqual(t, first.Header().Get("ETag"), last.Header().Get("ETag"))
}
