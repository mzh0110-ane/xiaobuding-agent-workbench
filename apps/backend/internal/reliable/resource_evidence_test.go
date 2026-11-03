package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeferredResourceReadRejected(t *testing.T) {
	ctx := context.Background()
	work, _, err := PrepareWorktree(ctx, t.TempDir(), resourceFixture(t), "read-resource")
	require.NoError(t, err)
	before, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	path := "static/figs/00000.wmf"
	require.Error(t, checkReads([]string{path}, before))
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(work, filepath.FromSlash(path))), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(work, filepath.FromSlash(path)), []byte("image\x00input"), 0600))
	after, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	require.Equal(t, before.Hash, after.Hash, "unchanged input hydration must not change logical source")
	require.NoError(t, checkReads([]string{path}, after), "verified available resources can be cited")
	require.NoError(t, os.Remove(filepath.Join(work, filepath.FromSlash(path))))
	_, err = TakeSnapshot(ctx, work)
	require.Error(t, err, "deleting an acknowledged materialized input must be detected")
}

type resourceRequestAgent struct{ calls int }

func (a *resourceRequestAgent) Run(_ context.Context, r RoleRequest) (*AgentResult, error) {
	a.calls++
	if a.calls == 1 {
		return &AgentResult{Session: "request-input", Output: []byte(`{"resource_requests":["static/figs/00000.wmf"]}`)}, nil
	}
	if _, err := os.Stat(filepath.Join(r.Directory, "static", "figs", "00000.wmf")); err != nil {
		return nil, err
	}
	return &AgentResult{Session: "after-input", Output: []byte(`{"summary":"input available","read_files":["app.txt","static/figs/00000.wmf"],"docs_impact":"none","residual_risks":[]}`)}, nil
}

func TestResourceRequestDispatch(t *testing.T) {
	agent := &resourceRequestAgent{}
	e, task := fixtureEngine(t, agent, passingCheck)
	project := resourceFixture(t)
	var err error
	task.Project = project
	task.Worktree, task.Base, err = PrepareWorktree(context.Background(), e.Root, project, task.ID)
	require.NoError(t, err)
	task.Baseline, err = TakeSnapshot(context.Background(), task.Worktree)
	require.NoError(t, err)
	_, err = e.Store.Update(task.ID, func(saved *Task) error {
		saved.Project = project
		saved.Worktree = task.Worktree
		saved.Base = task.Base
		saved.Baseline = task.Baseline
		return nil
	})
	require.NoError(t, err)
	result, err := e.call(context.Background(), task, roleImplementer, "read input", 1)
	require.NoError(t, err)
	var output map[string]any
	require.NoError(t, json.Unmarshal(result.Output, &output))
	require.Equal(t, "input available", output["summary"])
	require.Equal(t, 2, agent.calls)
	saved, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Zero(t, saved.Round)
	require.Len(t, saved.Runs, 2)
}

func TestResourceRequestProtocol(t *testing.T) {
	for _, raw := range []string{
		`{"resource_requests":["x"],"read_files":["invented.png"]}`,
		`{"resource_requests":["x"],"verdict":"pass"}`,
		`{"resource_requests":"x"}`,
		`{"resource_requests":["x"],"summary":"completed all checks"}`,
	} {
		_, _, err := resourceRequestOutput([]byte(raw))
		require.Error(t, err)
	}
	paths, raw, err := resourceRequestOutput([]byte(`{"resource_requests":[],"summary":"normal"}`))
	require.NoError(t, err)
	require.Empty(t, paths)
	require.NotContains(t, string(raw), "resource_requests")
	for _, role := range []string{roleAuditor, roleImplementer, roleReviewer} {
		var schema map[string]any
		require.NoError(t, json.Unmarshal([]byte(schemaForRequest(RoleRequest{Role: role, ResourceMode: true})), &schema))
		require.Contains(t, schema["properties"], "resource_requests")
		require.Contains(t, schema["required"], "resource_requests")
	}
}
