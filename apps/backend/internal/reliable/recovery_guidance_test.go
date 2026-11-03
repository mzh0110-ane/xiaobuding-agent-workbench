package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoveryHTTPClassifiesReasonWithoutMutatingTask(t *testing.T) {
	cases := []struct{ reason, code, action string }{
		{"codex auditor failed: context deadline exceeded", "execution_timeout", "resume"},
		{"stage_resume_source_changed: change outside audited scope: docs/a.md", "source_changed", "replan"},
		{"authentication failed: login expired", "login_required", "login"},
		{"three rounds exhausted: missing behavior", "round_limit", "new_batch"},
		{"A required verification command failed", "checks_failed", "checks"},
		{"resource_request: invalid request", "evidence_invalid", "none"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
			_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.State, saved.Reason = stateBlocked, tc.reason; return nil })
			if err != nil {
				t.Fatal(err)
			}
			h := &HTTP{Engine: e, Token: "token"}
			response := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/recovery", "", "")
			if response.Code != http.StatusOK {
				t.Fatalf("recovery guidance not exposed: %d %s", response.Code, response.Body)
			}
			var got struct {
				Code, Summary, Detail string
				NextAction            string `json:"next_action"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Code != tc.code || got.NextAction != tc.action || got.Summary == "" || got.Detail != tc.reason {
				t.Fatalf("wrong recovery instruction: %+v", got)
			}
			saved, _ := e.Store.Get(task.ID)
			if saved.State != stateBlocked || saved.Reason != tc.reason || saved.Round != 0 {
				t.Fatal("guidance changed the task")
			}
		})
	}
}

func TestRecoveryGuidanceUsesTaskLifecycleBeforeHistoricalReason(t *testing.T) {
	cases := []struct{ name, state, reason, code, action string }{
		{"exhausted", stateExhausted, "Reviewer found an unmet requirement", "round_limit", "new_batch"},
		{"accepted", stateAccepted, "context deadline exceeded", "none", "none"},
		{"awaiting", stateAwaitingAcceptance, "old timeout", "none", "none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
			_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.State, saved.Reason = tc.state, tc.reason; return nil })
			if err != nil {
				t.Fatal(err)
			}
			h := &HTTP{Engine: e, Token: "token"}
			response := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/recovery", "", "")
			var got struct {
				Code       string
				NextAction string `json:"next_action"`
			}
			if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &got) != nil {
				t.Fatalf("guidance unavailable: %s", response.Body)
			}
			if got.Code != tc.code || got.NextAction != tc.action {
				t.Fatalf("historical reason created action: %+v", got)
			}
		})
	}
}

type auditRetryFixture struct {
	base            fixtureAgent
	calls, failures int
	mutate          bool
	callback        func()
	err             error
}

func (a *auditRetryFixture) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	if request.Role != roleAuditor {
		return a.base.Run(ctx, request)
	}
	a.calls++
	if a.failures <= 0 {
		return a.base.Run(ctx, request)
	}
	a.failures--
	if a.mutate {
		_ = os.WriteFile(filepath.Join(request.Directory, "app.txt"), []byte("unexpected\n"), 0600)
	}
	if a.callback != nil {
		a.callback()
	}
	return nil, a.err
}

func preparedAuditRetry(t *testing.T, agent *auditRetryFixture) (*Engine, *Task, context.Context) {
	t.Helper()
	e, task := fixtureEngine(t, agent, passingCheck)
	ctx, _ := observeOwnedCleanup(context.Background())
	if err := e.prepare(ctx, task); err != nil {
		t.Fatal(err)
	}
	return e, task, ctx
}

func TestAuditRetriesTransientInvocationWithoutSpendingQualityRound(t *testing.T) {
	agent := &auditRetryFixture{failures: 1, err: context.DeadlineExceeded}
	e, task, ctx := preparedAuditRetry(t, agent)
	if err := e.audit(ctx, task); err != nil {
		t.Fatalf("transient planning failure was not recovered: %v", err)
	}
	saved, _ := e.Store.Get(task.ID)
	if agent.calls != 2 || saved.Round != 0 || saved.Audit == nil || len(saved.Runs) != 2 {
		t.Fatalf("retry lost evidence or spent a round: calls=%d task=%+v", agent.calls, saved)
	}
}

func TestAuditRuntimeRetryIsBoundedAndRejectsSourceOrIdentityChanges(t *testing.T) {
	cases := []struct {
		name                    string
		mutate, revision, model bool
		err                     error
		want                    int
	}{
		{"bounded", false, false, false, context.DeadlineExceeded, 3},
		{"source", true, false, false, context.DeadlineExceeded, 1},
		{"requirements", false, true, false, context.DeadlineExceeded, 1},
		{"models", false, false, true, context.DeadlineExceeded, 1},
		{"nontransient", false, false, false, errors.New("invalid agent schema"), 1},
		{"cancelled", false, false, false, context.Canceled, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agent := &auditRetryFixture{failures: 4, mutate: tc.mutate, err: tc.err}
			e, task, ctx := preparedAuditRetry(t, agent)
			if tc.revision {
				agent.callback = func() {
					_, _ = e.Store.Update(task.ID, func(saved *Task) error { saved.RequirementRevision++; return nil })
				}
			}
			if tc.model {
				agent.callback = func() { _, _ = e.Store.Update(task.ID, func(saved *Task) error { saved.PlanRevision++; return nil }) }
			}
			if err := e.audit(ctx, task); err == nil {
				t.Fatal("invalid audit completed")
			}
			saved, _ := e.Store.Get(task.ID)
			if agent.calls != tc.want || saved.Round != 0 {
				t.Fatalf("unsafe planning retry: calls=%d round=%d", agent.calls, saved.Round)
			}
		})
	}
}

func TestAuditReportValidationIsNotRetriedAsNetworkFailure(t *testing.T) {
	agent := &auditRetryFixture{err: errors.New("not used")}
	e, task, ctx := preparedAuditRetry(t, agent)
	e.Runner = agentFunc(func(_ context.Context, _ RoleRequest) (*AgentResult, error) {
		agent.calls++
		return &AgentResult{Session: "schema-session", Output: json.RawMessage(`{"summary":"timeout is mentioned in a malformed report"}`)}, nil
	})
	err := e.audit(ctx, task)
	if err == nil || agent.calls != 1 || strings.Contains(err.Error(), "planning runtime retry") {
		t.Fatalf("report failure retried: %v calls=%d", err, agent.calls)
	}
}
