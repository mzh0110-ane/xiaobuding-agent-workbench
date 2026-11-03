package reliable

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type auditResumeGuardAgent struct {
	base     fixtureAgent
	calls    int
	mutate   bool
	firstErr error
}

func (agent *auditResumeGuardAgent) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	if request.Role != roleAuditor {
		return agent.base.Run(ctx, request)
	}
	agent.calls++
	if agent.calls != 1 {
		return agent.base.Run(ctx, request)
	}
	if agent.mutate {
		if err := os.WriteFile(filepath.Join(request.Directory, "app.txt"), []byte("untrusted-audit-change\n"), 0600); err != nil {
			return nil, err
		}
	}
	if agent.firstErr != nil {
		return nil, agent.firstErr
	}
	return agent.base.Run(ctx, request)
}

func TestAuditManualResumeRejectsChangedFailedAndCompletedInvocations(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"failed_invocation", context.DeadlineExceeded},
		{"completed_invocation", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agent := &auditResumeGuardAgent{mutate: true, firstErr: tc.err}
			engine, task := fixtureEngine(t, agent, passingCheck)
			if err := engine.RunTask(context.Background(), task.ID); err == nil {
				t.Fatal("mutating read-only audit was accepted")
			}
			before, err := engine.Store.Get(task.ID)
			if err != nil || before.Round != 0 || agent.calls != 1 {
				t.Fatalf("initial mutation was not stopped before implementation: %v calls=%d", err, agent.calls)
			}
			err = engine.RunTask(context.Background(), task.ID)
			if err == nil || !strings.Contains(err.Error(), "audit_resume_source_changed") {
				t.Fatalf("manual resume trusted the changed source: %v calls=%d", err, agent.calls)
			}
			after, err := engine.Store.Get(task.ID)
			if err != nil || after.Round != 0 || agent.calls != 1 || len(after.Runs) != len(before.Runs) || after.State != stateBlocked {
				t.Fatalf("rejected recovery changed execution history: %v calls=%d", err, agent.calls)
			}
			raw, err := os.ReadFile(filepath.Join(after.Worktree, "app.txt"))
			if err != nil || string(raw) != "untrusted-audit-change\n" {
				t.Fatalf("recovery rewrote retained source: %q %v", raw, err)
			}
			guidance := RecoveryGuidanceFor(after)
			if guidance.Code != "source_changed" || guidance.NextAction != "replan" {
				t.Fatalf("source drift did not receive explicit recovery guidance: %+v", guidance)
			}
		})
	}
}

func TestAuditManualResumeAllowsUnchangedSourceAfterInvocationFailure(t *testing.T) {
	agent := &auditResumeGuardAgent{firstErr: errors.New("native planning service unavailable")}
	engine, task := fixtureEngine(t, agent, passingCheck)
	if err := engine.RunTask(context.Background(), task.ID); err == nil {
		t.Fatal("positive-control invocation did not fail")
	}
	if err := engine.RunTask(context.Background(), task.ID); err != nil {
		t.Fatalf("unchanged candidate could not resume: %v", err)
	}
	saved, err := engine.Store.Get(task.ID)
	if err != nil || saved.Round != 1 || saved.State != stateAwaitingAcceptance || agent.calls != 2 {
		t.Fatalf("clean recovery did not reach independent review: %v calls=%d", err, agent.calls)
	}
}
