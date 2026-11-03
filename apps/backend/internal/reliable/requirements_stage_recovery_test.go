package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequirementsStageRecoveryFailedAuditRetriesFreshPlan(t *testing.T) {
	engine, paused, base, agent := requirementsStagePause(t)
	oldAudit, _ := json.Marshal(paused.Audit)
	oldStage, oldEvidence := *paused.StageRecovery, append([]string(nil), paused.DomainEvidence...)
	evidencePath := filepath.Join(engine.Root, "evidence", paused.ID, "round-1-frontend-implementation.json")
	oldBytes, err := os.ReadFile(evidencePath)
	require.NoError(t, err)
	http := &HTTP{Engine: engine, Token: "frozen-audit-requirement"}
	attachment := flowUploadAttachment(t, http, "requirement.txt", []byte("Retain the interface"))
	_, err = engine.submitMessage(paused.ID, messageRequest{ID: "new-scope", Kind: messageRequirement, Content: "Retain the interface", AttachmentIDs: []string{attachment["id"].(string)}})
	require.NoError(t, err)
	requirementsStageAuditFailure(agent)
	require.Error(t, engine.RunTask(context.Background(), paused.ID))
	failed := requirementsStageSaved(t, engine, paused.ID)
	require.Equal(t, 1, failed.RequirementRevision)
	require.Zero(t, failed.AuditedRevision)
	require.Equal(t, stateQueued, failed.Messages[0].Status)
	require.Zero(t, failed.Messages[0].AppliedRound)
	require.False(t, failed.ContinuePending, "the failed audit consumed its automatic continuation request")
	require.Equal(t, 1, failed.Round)
	require.Equal(t, oldStage, *failed.StageRecovery)
	require.Equal(t, oldEvidence, failed.DomainEvidence)
	actualAudit, _ := json.Marshal(failed.Audit)
	require.Equal(t, oldAudit, actualAudit)
	actualBytes, err := os.ReadFile(evidencePath)
	require.NoError(t, err)
	require.Equal(t, oldBytes, actualBytes)
	require.False(t, domainStageResumeEligible(failed))
	input := requirementsStageHandoff(t, engine, failed)
	require.Equal(t, 1, input.RequirementRevision)
	require.Contains(t, input.Goal, "USER REQUIREMENT 1")
	require.Len(t, input.Attachments, 1)
	require.Equal(t, attachment["id"], input.Attachments[0].ID)
	agent.failure = nil
	require.NoError(t, engine.RunTask(context.Background(), paused.ID))
	finished := requirementsStageSaved(t, engine, paused.ID)
	require.Equal(t, 3, agent.audits)
	require.Equal(t, stateAwaitingAcceptance, finished.State)
	require.Equal(t, 2, finished.Round)
	require.Equal(t, 1, finished.AuditedRevision)
	require.Equal(t, "applied", finished.Messages[0].Status)
	require.Equal(t, 2, finished.Messages[0].AppliedRound)
	require.Equal(t, 2, base.frontend)
	require.Equal(t, 1, base.backend)
	require.Equal(t, 1, base.reviews)
	require.NotEmpty(t, finished.Checks)
	require.NotEmpty(t, finished.Review)
}

func TestRequirementsStageRecoveryInvalidAuditDoesNotPromote(t *testing.T) {
	engine, task, agent := requirementsStagePrepared(t)
	_, err := engine.Store.Update(task.ID, func(saved *Task) error {
		saved.AllowedPaths = []string{"app.txt"}
		return nil
	})
	require.NoError(t, err)
	old := requirementsStageSaved(t, engine, task.ID)
	for _, output := range []struct{ name, raw string }{
		{"invalid-json", `{"summary":`},
		{"incomplete", `{"summary":"Incomplete"}`},
		{"scope-invalid", `{"summary":"Unsafe scope","read_files":["app.txt"],"required_read_files":["app.txt"],"allowed_paths":["outside.txt"],"plan":["Update outside.txt"]}`},
	} {
		t.Run(output.name, func(t *testing.T) {
			agent.output = output.raw
			require.Error(t, engine.audit(context.Background(), task))
			current := requirementsStageSaved(t, engine, task.ID)
			require.Equal(t, old.Audit, current.Audit)
			require.Equal(t, old.ApprovedPaths, current.ApprovedPaths)
			require.Zero(t, current.AuditedRevision)
			require.Equal(t, 1, current.Round)
			require.Equal(t, stateQueued, current.Messages[0].Status)
			require.Zero(t, current.Messages[0].AppliedRound)
		})
	}
}

func TestRequirementsStageRecoveryLegacyPromotedRevisionReaudits(t *testing.T) {
	engine, paused, base, agent := requirementsStagePause(t)
	requirementsStageSubmit(t, engine, paused.ID, "new-scope")
	_, err := engine.Store.Update(paused.ID, func(saved *Task) error {
		saved.AuditedRevision = saved.RequirementRevision
		saved.Messages[0].Status, saved.Messages[0].AppliedRound = "applied", saved.Round+1
		return nil
	})
	require.NoError(t, err)
	legacy := requirementsStageSaved(t, engine, paused.ID)
	require.ErrorContains(t, validateStageIdentity(legacy), "stage_resume_invalidated")
	require.False(t, domainStageResumeEligible(legacy))
	require.NoError(t, engine.RunTask(context.Background(), paused.ID))
	finished := requirementsStageSaved(t, engine, paused.ID)
	require.Equal(t, 2, agent.audits)
	require.Equal(t, 2, finished.Round)
	require.Equal(t, stateAwaitingAcceptance, finished.State)
	require.Equal(t, 2, base.frontend)
	require.Equal(t, 1, base.reviews)
}
