package reliable

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func modelRequest(t *testing.T, h *HTTP, id, action string, revision int, plan ModelPlan) int {
	t.Helper()
	body, err := json.Marshal(map[string]any{"expected_revision": revision, "plan": plan})
	require.NoError(t, err)
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+id+"/"+action, string(body), h.Token)
	if response.Code >= 400 {
		t.Logf("%s: %d %s", action, response.Code, response.Body.String())
	}
	return response.Code
}

func TestCompletedInspectionCanChooseTheNextQuestionModelAndEffort(t *testing.T) {
	probe := &conversationProbe{}
	e, task := fixtureEngine(t, probe, passingCheck)
	t.Cleanup(e.wg.Wait)
	e.Providers = []Provider{{ID: "codex", Name: "Codex", Model: "native-sol", Available: true, Models: []ModelOption{{ID: "native-sol", Efforts: []string{"low", "ultra"}, DefaultEffort: "low"}}}}
	plan := ModelPlan{Auditor: "codex", Implementer: "codex", Reviewer: "codex"}
	_, err := e.UpdateModelPlan(task.ID, 0, plan)
	require.NoError(t, err)
	_, err = e.Store.Update(task.ID, func(saved *Task) error { saved.State, saved.ReadOnly = stateInspected, true; return nil })
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "future-plan-test"}
	plan.AuditorModel, plan.AuditorEffort = "native-sol", "ultra"
	require.Equal(t, http.StatusOK, modelRequest(t, h, task.ID, "future-models", 0, plan))
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"q1","kind":"question","content":"Explain the current project"}`, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code)
	e.wg.Wait()
	require.Len(t, probe.requests, 1)
	require.Equal(t, "native-sol", probe.requests[0].Model)
	require.Equal(t, "ultra", probe.requests[0].Effort)
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Empty(t, current.ModelPlan.AuditorEffort, "past execution plan was rewritten")
	require.Equal(t, 1, current.PlanRevision)
}

func TestFutureSettingsPreservePassingEvidenceAndVersionGuards(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	e.Providers = testProviders()
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	before, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "sealed-future-test"}
	plan := e.defaultPlan()
	plan.Auditor = "codex"
	require.Equal(t, http.StatusOK, modelRequest(t, h, task.ID, "future-models", 0, plan))
	after, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, before.ModelPlan, after.ModelPlan)
	require.Equal(t, before.PlanEvidence, after.PlanEvidence)
	require.Equal(t, before.Runs, after.Runs)
	require.Equal(t, before.EvidenceHash, after.EvidenceHash)
	_, err = e.acceptanceEvidence(context.Background(), after)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, modelRequest(t, h, task.ID, "future-models", 0, plan))
	require.NoError(t, e.ArchiveTask(task.ID, true))
	require.Equal(t, http.StatusConflict, modelRequest(t, h, task.ID, "future-models", 1, plan))
}

func TestFutureChoicesReachLinkedWorkAndReworkEvidence(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(e.wg.Wait)
	e.Providers = testProviders()
	initial := ModelPlan{Auditor: "deepseek", Implementer: "codex", Reviewer: "deepseek"}
	_, err := e.UpdateModelPlan(task.ID, 0, initial)
	require.NoError(t, err)
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	h := &HTTP{Engine: e, Token: "future-rework"}
	changed := ModelPlan{Auditor: "codex", Implementer: "deepseek", Reviewer: "codex"}
	require.Equal(t, http.StatusOK, modelRequest(t, h, task.ID, "future-models", 0, changed))
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"r1","kind":"requirement","content":"Also retain the public interface"}`, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	e.wg.Wait()
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateAwaitingAcceptance, current.State, current.Reason)
	require.Equal(t, 2, current.Round)
	require.Equal(t, changed, current.ModelPlan)
	require.Nil(t, current.FuturePlan)
	require.Len(t, current.PlanEvidence, 2)
	_, err = e.acceptanceEvidence(context.Background(), current)
	require.NoError(t, err)
	for _, run := range current.Runs[len(current.Runs)-3:] {
		require.Equal(t, 2, run.PlanRevision)
	}
	require.NoError(t, e.Accept(context.Background(), task.ID))
	require.Equal(t, http.StatusOK, modelRequest(t, h, task.ID, "future-models", 0, initial))
	message, err := e.startLinkedBatch(task.ID, messageRequest{ID: "batch-2", Kind: messageRequirement, Content: "Next feature"})
	require.NoError(t, err)
	child, err := e.Store.Get(message.LinkedTask)
	require.NoError(t, err)
	require.Equal(t, initial, child.ModelPlan)
}

func TestChangingFutureEffortKeepsTheActiveQuestionFrozen(t *testing.T) {
	probe := &conversationProbe{started: make(chan struct{}), release: make(chan struct{})}
	e, task := fixtureEngine(t, probe, passingCheck)
	t.Cleanup(e.wg.Wait)
	e.Providers = []Provider{{ID: "codex", Name: "Codex", Model: "native-sol", Available: true, Models: []ModelOption{{ID: "native-sol", Efforts: []string{"low", "ultra"}, DefaultEffort: "low"}}}}
	initial := ModelPlan{Auditor: "codex", Implementer: "codex", Reviewer: "codex"}
	_, err := e.UpdateModelPlan(task.ID, 0, initial)
	require.NoError(t, err)
	_, err = e.Store.Update(task.ID, func(saved *Task) error { saved.State, saved.ReadOnly = stateInspected, true; return nil })
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "future-question-gate"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"q1","kind":"question","content":"First question"}`, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code)
	select {
	case <-probe.started:
	case <-time.After(5 * time.Second):
		t.Fatal("question did not start")
	}
	changed := initial
	changed.AuditorEffort = "ultra"
	status := modelRequest(t, h, task.ID, "future-models", 0, changed)
	response = httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"q2","kind":"question","content":"Second question"}`, h.Token)
	close(probe.release)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, http.StatusAccepted, response.Code)
	e.wg.Wait()
	require.Len(t, probe.requests, 2)
	require.Equal(t, "low", probe.requests[0].Effort)
	require.Equal(t, "ultra", probe.requests[1].Effort)
}
