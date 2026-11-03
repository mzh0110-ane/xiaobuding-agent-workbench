package reliable

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"path/filepath"
	"testing"
)

func TestReadOnlyContinuationCanChooseSpecificProject(t *testing.T) {
	e, parent := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	_, err := e.Store.Update(parent.ID, func(t *Task) error {
		t.ReadOnly = true
		t.State = stateInspected
		t.Audit = &Audit{Summary: "existing investigation"}
		t.Commands = []string{"wrong project command"}
		t.AllowedPaths = []string{"other/path"}
		return nil
	})
	require.NoError(t, err)
	target := t.TempDir()
	request := messageRequest{ID: "next-project", Kind: messageRequirement, Content: "Make a change", Project: target}
	m, err := e.startLinkedBatch(parent.ID, request)
	require.NoError(t, err)
	child, err := e.Store.Get(m.LinkedTask)
	require.NoError(t, err)
	resolved, err := filepath.EvalSymlinks(target)
	require.NoError(t, err)
	require.Equal(t, resolved, child.Project)
	require.Equal(t, parent.ID, child.ParentTaskID)
	require.NotContains(t, child.Commands, "wrong project command")
	require.NotContains(t, child.AllowedPaths, "other/path")
	var ctx map[string]any
	require.NoError(t, json.Unmarshal(child.PreviousContext, &ctx))
	require.Contains(t, string(child.PreviousContext), "existing investigation")
	request.Project = t.TempDir()
	_, err = e.startLinkedBatch(parent.ID, request)
	require.Error(t, err)
}

func TestNonReadOnlyContinuationRejectsChangedProject(t *testing.T) {
	e, parent := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	_, err := e.Store.Update(parent.ID, func(t *Task) error { t.State = stateAccepted; return nil })
	require.NoError(t, err)
	_, err = e.startLinkedBatch(parent.ID, messageRequest{ID: "different", Kind: messageRequirement, Content: "Change", Project: t.TempDir()})
	require.Error(t, err)
}

func TestHTTPReadOnlyContinuationIncludesSelectedProject(t *testing.T) {
	e, parent := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(e.wg.Wait)
	_, err := e.Store.Update(parent.ID, func(task *Task) error { task.ReadOnly = true; task.State = stateInspected; return nil })
	require.NoError(t, err)
	target := t.TempDir()
	body, err := json.Marshal(map[string]string{"id": "target-project", "kind": "requirement", "content": "Implement a change", "project": target})
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "readonly-project-target"}
	result := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+parent.ID+"/messages", string(body), h.Token)
	require.Equal(t, http.StatusAccepted, result.Code, result.Body.String())
	var m TaskMessage
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &m))
	child, err := e.Store.Get(m.LinkedTask)
	require.NoError(t, err)
	resolved, err := filepath.EvalSymlinks(target)
	require.NoError(t, err)
	require.Equal(t, resolved, child.Project)
	require.Equal(t, resolved, m.TargetProject)
}

func TestHTTPRejectsProjectOverrideOnOrdinaryTask(t *testing.T) {
	e, parent := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	body, err := json.Marshal(map[string]string{"id": "wrong-target", "kind": "requirement", "content": "Make a change", "project": t.TempDir()})
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "readonly-project-target"}
	result := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+parent.ID+"/messages", string(body), h.Token)
	require.NotEqual(t, http.StatusAccepted, result.Code)
}
