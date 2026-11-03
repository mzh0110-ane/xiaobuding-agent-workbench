package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func replanSeedRequirementHistory(t *testing.T, revisions int) (*Engine, *Task) {
	t.Helper()
	root := t.TempDir()
	store, err := OpenStore(filepath.Join(root, "state.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	engine := &Engine{Store: store, Root: root}
	task := &Task{ID: "replan-seed-history", Project: t.TempDir(), Worktree: t.TempDir(), Base: "original-base", State: stateBlocked, AllowedPaths: []string{"app.txt"}}
	require.NoError(t, store.Create(task))
	for i := 0; i < max(1, revisions); i++ {
		if revisions > 0 {
			requirementsStageSubmit(t, engine, task.ID, fmt.Sprintf("requirement-%d", i+1))
		}
		task, err = store.Update(task.ID, func(saved *Task) error { saved.State = stateBlocked; return nil })
		require.NoError(t, err)
		_, err = engine.saveAuditReplanApproval(context.Background(), task, auditReplanRequest{SourceHash: digest([]byte(fmt.Sprint(i))), Confirm: true})
		require.NoError(t, err)
	}
	return engine, requirementsStageSaved(t, engine, task.ID)
}

func TestAuditReplanRecoveryRequirementsAcceptsAuthenticatedInterleaving(t *testing.T) {
	for _, revisions := range []int{0, 2} {
		t.Run(fmt.Sprint(revisions), func(t *testing.T) {
			engine, task := replanSeedRequirementHistory(t, revisions)
			require.Equal(t, revisions, task.RequirementRevision)
			require.Greater(t, len(task.RequirementEvidence), revisions)
			require.NoError(t, engine.recoveryRequirements(task))
		})
	}
}

func TestAuditReplanRecoveryRequirementsRejectsChangedEvidence(t *testing.T) {
	tests := []struct {
		name string
		edit func(*testing.T, *Engine, *Task)
	}{
		{"latest-receipt-changed", func(t *testing.T, e *Engine, task *Task) {
			rewriteHumanReceipt(t, e, task, task.RequirementEvidence[3])
		}},
		{"historical-receipt-changed", func(t *testing.T, e *Engine, task *Task) {
			rewriteHumanReceipt(t, e, task, task.RequirementEvidence[1])
		}},
		{"latest-receipt-missing", func(t *testing.T, e *Engine, task *Task) { replanSeedRemoveEvidence(t, e, task, 3) }},
		{"historical-receipt-missing", func(t *testing.T, e *Engine, task *Task) { replanSeedRemoveEvidence(t, e, task, 1) }},
		{"unapproved-prefix", func(_ *testing.T, _ *Engine, task *Task) {
			task.RequirementEvidence = append(task.RequirementEvidence, "audit-replan-unapproved.json")
		}},
		{"approval-missing", func(_ *testing.T, _ *Engine, task *Task) { task.AuditReplan = nil }},
		{"receipt-reference-missing", func(_ *testing.T, _ *Engine, task *Task) { task.RequirementEvidence = task.RequirementEvidence[:3] }},
		{"receipt-reference-duplicated", func(_ *testing.T, _ *Engine, task *Task) {
			task.RequirementEvidence = append(task.RequirementEvidence, task.AuditReplan.Receipt)
		}},
		{"requirement-reference-missing", func(_ *testing.T, _ *Engine, task *Task) { task.RequirementEvidence = task.RequirementEvidence[1:] }},
		{"requirement-reordered", func(_ *testing.T, _ *Engine, task *Task) {
			task.RequirementEvidence[0], task.RequirementEvidence[2] = task.RequirementEvidence[2], task.RequirementEvidence[0]
		}},
		{"requirement-missing", func(t *testing.T, e *Engine, task *Task) { replanSeedRemoveEvidence(t, e, task, 0) }},
		{"requirement-message-changed", func(_ *testing.T, _ *Engine, task *Task) { task.Messages[0].Content = "Changed requirement" }},
		{"requirement-count-changed", func(_ *testing.T, _ *Engine, task *Task) { task.RequirementRevision++ }},
		{"unknown-evidence", func(t *testing.T, e *Engine, task *Task) {
			require.NoError(t, e.saveJSON(task.ID, "unknown.json", map[string]bool{"fixture": true}))
			task.RequirementEvidence = append(task.RequirementEvidence, "unknown.json")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, task := replanSeedRequirementHistory(t, 2)
			require.NoError(t, engine.validateAuditReplanEvidence(task, nil))
			test.edit(t, engine, task)
			require.Error(t, engine.recoveryRequirements(task))
		})
	}
}

func replanSeedRemoveEvidence(t *testing.T, engine *Engine, task *Task, index int) {
	t.Helper()
	require.NoError(t, os.Remove(filepath.Join(engine.Root, "evidence", task.ID, task.RequirementEvidence[index])))
}

type replanSeedInterruptedAgent struct {
	base          fixtureAgent
	writerFailed  bool
	failNextAudit bool
}

func (agent *replanSeedInterruptedAgent) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	if request.Role == roleAuditor && agent.failNextAudit {
		agent.failNextAudit = false
		return nil, errors.New("fixture planning interrupted")
	}
	if request.Role == roleImplementer && !agent.writerFailed {
		agent.writerFailed = true
		if err := os.WriteFile(filepath.Join(request.Directory, "app.txt"), []byte("retained writer progress\n"), 0600); err != nil {
			return nil, err
		}
		return nil, errors.New("fixture writer interrupted")
	}
	return agent.base.Run(ctx, request)
}

func TestAuditReplanInheritedSeedResumesThroughVerification(t *testing.T) {
	agent := &replanSeedInterruptedAgent{}
	engine, parent := fixtureEngine(t, agent, passingCheck)
	t.Cleanup(func() { _ = engine.Close() })
	parent, err := engine.Store.Update(parent.ID, func(saved *Task) error { saved.AllowedPaths = []string{"app.txt"}; return nil })
	require.NoError(t, err)
	require.NoError(t, engine.prepare(context.Background(), parent))
	parent, err = engine.Store.Update(parent.ID, func(saved *Task) error {
		saved.State, saved.Round, saved.Reason = stateExhausted, MaxRounds, "Review findings remain unresolved"
		saved.ApprovedPaths = []string{"app.txt"}
		return nil
	})
	require.NoError(t, err)
	child := claimExhaustedChild(t, engine, parent, "inherited-replan")
	require.ErrorContains(t, engine.RunTask(context.Background(), child.ID), "fixture writer interrupted")
	requirementsStageSubmit(t, engine, child.ID, "new-scope")
	agent.failNextAudit = true
	require.ErrorContains(t, engine.RunTask(context.Background(), child.ID), "fixture planning interrupted")
	stopped := requirementsStageSaved(t, engine, child.ID)
	require.Equal(t, 1, stopped.Round)
	require.Len(t, stopped.SeedEvidence, 2)
	require.Empty(t, stopped.EvidenceHash)
	handler := &HTTP{Engine: engine, Token: "inherited-replan-confirmation"}
	inspection := replanGet(t, handler, stopped)
	require.Equal(t, true, inspection["eligible"])
	body, err := json.Marshal(map[string]any{"source_hash": inspection["source_hash"], "confirm": true})
	require.NoError(t, err)
	response := httpRequest(handler, http.MethodPost, "/api/v1/reliable/tasks/"+child.ID+"/replan", string(body), handler.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	engine.wg.Wait()
	finished := requirementsStageSaved(t, engine, child.ID)
	require.Equal(t, stateAwaitingAcceptance, finished.State, finished.Reason)
	require.Equal(t, 2, finished.Round)
	require.Equal(t, stopped.Worktree, finished.Worktree)
	require.Equal(t, stopped.SeedEvidence, finished.SeedEvidence)
	require.Equal(t, 1, finished.RequirementRevision)
	require.Len(t, finished.RequirementEvidence, 2)
	require.NoError(t, engine.recoveryRequirements(finished))
	require.NotEmpty(t, finished.Checks)
	require.NotEmpty(t, finished.Review)
	require.Equal(t, 1, agent.base.review)
	_, err = engine.acceptanceEvidence(context.Background(), finished)
	require.NoError(t, err)
}
