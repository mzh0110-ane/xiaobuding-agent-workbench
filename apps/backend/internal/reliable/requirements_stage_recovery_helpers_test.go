package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type requirementsStageAuditFixture struct {
	base        AgentRunner
	audits      int
	failure     error
	output      string
	beforeAudit func(RoleRequest)
}

func (a *requirementsStageAuditFixture) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	if request.Role == roleAuditor {
		a.audits++
		if a.beforeAudit != nil {
			a.beforeAudit(request)
		}
		if a.failure != nil {
			return nil, a.failure
		}
	}
	result, err := a.base.Run(ctx, request)
	if err == nil && request.Role == roleAuditor && a.output != "" {
		result.Output = json.RawMessage(a.output)
	}
	return result, err
}

func requirementsStageSaved(t *testing.T, engine *Engine, id string) *Task {
	t.Helper()
	task, err := engine.Store.Get(id)
	require.NoError(t, err)
	return task
}

func requirementsStageSubmit(t *testing.T, engine *Engine, id, messageID string) {
	t.Helper()
	_, err := engine.submitMessage(id, messageRequest{ID: messageID, Kind: messageRequirement, Content: "Keep the updated public interface: " + messageID})
	require.NoError(t, err)
}

func requirementsStagePrepared(t *testing.T) (*Engine, *Task, *requirementsStageAuditFixture) {
	t.Helper()
	agent := &requirementsStageAuditFixture{base: &fixtureAgent{}}
	engine, task := fixtureEngine(t, agent, passingCheck)
	require.NoError(t, engine.prepare(context.Background(), task))
	require.NoError(t, engine.audit(context.Background(), task))
	current, err := engine.Store.StartRound(task.ID)
	require.NoError(t, err)
	requirementsStageSubmit(t, engine, task.ID, "new-scope")
	return engine, current, agent
}

func requirementsStagePause(t *testing.T) (*Engine, *Task, *domainFixture, *requirementsStageAuditFixture) {
	t.Helper()
	base := &domainFixture{}
	engine, task := domainEngine(t, base, domainCheck)
	agent := &requirementsStageAuditFixture{base: &stageResumeFixture{base: base, failures: maxStageAttempts}}
	engine.Runner = agent
	require.Error(t, engine.RunTask(context.Background(), task.ID))
	current := requirementsStageSaved(t, engine, task.ID)
	require.Equal(t, 1, current.Round)
	require.True(t, domainStageResumeEligible(current))
	return engine, current, base, agent
}

func requirementsStageAuditFailure(agent *requirementsStageAuditFixture) {
	agent.failure = errors.New("fixture auditor unavailable")
}

func requirementsStageHandoff(t *testing.T, engine *Engine, task *Task) handoffInput {
	t.Helper()
	run := task.Runs[len(task.Runs)-1]
	require.Equal(t, roleAuditor, run.Role)
	var input handoffInput
	require.NoError(t, engine.readStageEvidence(task, run.InputEvidence, &input), fmt.Sprint(run))
	return input
}
