package reliable

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequirementsStageRecoveryConcurrentRequirementNeedsAnotherAudit(t *testing.T) {
	engine, task, agent := requirementsStagePrepared(t)
	_, err := engine.Store.Update(task.ID, func(saved *Task) error {
		saved.ApprovedPaths = append(saved.ApprovedPaths, "previously-approved.txt")
		return nil
	})
	require.NoError(t, err)
	agent.beforeAudit = func(request RoleRequest) {
		current := requirementsStageSaved(t, engine, task.ID)
		require.Zero(t, current.AuditedRevision)
		require.Equal(t, stateQueued, current.Messages[0].Status)
		require.Contains(t, request.Prompt, `"requirement_revision":1`)
		require.Contains(t, request.Prompt, "new-scope")
		require.NotContains(t, request.Prompt, "arrived-during-audit")
		requirementsStageSubmit(t, engine, task.ID, "arrived-during-audit")
	}
	require.NoError(t, engine.audit(context.Background(), task))
	current := requirementsStageSaved(t, engine, task.ID)
	require.Equal(t, 2, current.RequirementRevision)
	require.Equal(t, 1, current.AuditedRevision)
	require.Equal(t, "applied", current.Messages[0].Status)
	require.Equal(t, 2, current.Messages[0].AppliedRound)
	require.Equal(t, stateQueued, current.Messages[1].Status)
	require.Zero(t, current.Messages[1].AppliedRound)
	require.True(t, current.ContinuePending)
	require.Contains(t, current.ApprovedPaths, "previously-approved.txt")
	_, err = engine.Store.StartRound(task.ID)
	require.ErrorIs(t, err, errRequirementsChanged)
	agent.beforeAudit = nil
	require.NoError(t, engine.audit(context.Background(), task))
	current, err = engine.Store.StartRound(task.ID)
	require.NoError(t, err)
	require.Equal(t, 2, current.Round)
	require.Equal(t, 2, current.AuditedRevision)
	require.Equal(t, "applied", current.Messages[1].Status)
	require.Equal(t, 2, current.Messages[1].AppliedRound)
	require.False(t, current.ContinuePending)
}

func TestRequirementsStageRecoveryCancellationCannotCommitAudit(t *testing.T) {
	engine, task, agent := requirementsStagePrepared(t)
	old := requirementsStageSaved(t, engine, task.ID)
	agent.beforeAudit = func(RoleRequest) {
		_, err := engine.Store.Update(task.ID, func(saved *Task) error {
			saved.State = stateCancelling
			return nil
		})
		require.NoError(t, err)
	}
	require.ErrorIs(t, engine.audit(context.Background(), task), context.Canceled)
	current := requirementsStageSaved(t, engine, task.ID)
	require.Equal(t, stateCancelling, current.State)
	require.Equal(t, old.Audit, current.Audit)
	require.Equal(t, old.ApprovedPaths, current.ApprovedPaths)
	require.Zero(t, current.AuditedRevision)
	require.Equal(t, stateQueued, current.Messages[0].Status)
	require.Zero(t, current.Messages[0].AppliedRound)
}

func TestRequirementsStageRecoveryEligibilityRequiresCurrentIdentity(t *testing.T) {
	audit := &Audit{Summary: "Current audit"}
	raw, err := json.Marshal(audit)
	require.NoError(t, err)
	stage := &StageRecovery{Round: 3, Domain: domainBackend, Status: "retryable", AuditHash: digest(raw), RequirementRevision: 1}
	task := &Task{Round: 3, ModelPlan: ModelPlan{Mode: "fullstack"}, Audit: audit, Worktree: "retained", RequirementRevision: 1, AuditedRevision: 1, StageRecovery: stage}
	require.True(t, domainStageResumeEligible(task), "matching operational failure remains resumable on the third round")
	stage.RequirementRevision = 0
	require.False(t, domainStageResumeEligible(task))
	stage.RequirementRevision, stage.AuditHash = 1, "another-audit"
	require.False(t, domainStageResumeEligible(task))
}

func TestRequirementsStageRecoveryLegacyFallbackRejectsLaterAudit(t *testing.T) {
	task := &Task{Round: 1, ModelPlan: ModelPlan{Mode: "fullstack"}, Audit: &Audit{}, Worktree: "retained"}
	failed := Dispatch{Role: roleImplementer, Round: 1, Domain: domainBackend, Status: "failed"}
	task.Runs = []Dispatch{failed}
	require.True(t, domainStageResumeEligible(task))
	task.Runs = []Dispatch{{Role: roleAuditor, Round: 2, Status: "failed"}, failed}
	require.False(t, domainStageResumeEligible(task), "a historical fresh-audit attempt invalidates the legacy fallback")
	task.Runs[0].Status = dispatchNotStarted
	require.True(t, domainStageResumeEligible(task), "a dispatch that never started has no planning effect")
	task.RequirementRevision = 1
	require.False(t, domainStageResumeEligible(task), "queued requirements cannot reuse old writer inputs")
}
