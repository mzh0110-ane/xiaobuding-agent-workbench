package reliable

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestAssistantFutureSwitchKeepsTheSealedResultAndReachesNextBatch(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	e.Providers = testProviders()
	initial := assistantTestPlan(t, connectionCodex, connectionCodex)
	if _, err := e.UpdateModelPlan(task.ID, 0, initial); err != nil {
		t.Fatal(err)
	}
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	before, _ := e.Store.Get(task.ID)
	changed := assistantTestPlan(t, connectionHarness, connectionHarness)
	h := &HTTP{Engine: e, Token: "assistant-future-switch"}
	if code := modelRequest(t, h, task.ID, "future-models", 0, changed); code != http.StatusOK {
		t.Fatalf("future selection rejected: %d", code)
	}
	after, _ := e.Store.Get(task.ID)
	if after.ModelPlan != before.ModelPlan || after.Round != before.Round || after.Worktree != before.Worktree || after.EvidenceHash != before.EvidenceHash || !reflect.DeepEqual(after.Runs, before.Runs) {
		t.Fatal("future assistant selection rewrote reviewed work")
	}
	if code := modelRequest(t, h, task.ID, "future-models", 0, initial); code != http.StatusConflict {
		t.Fatal("stale assistant choice overwrote a newer selection")
	}
	question := &TaskMessage{}
	if err := e.questionSelection(after, question); err != nil || question.Provider != connectionHarness {
		t.Fatalf("next question did not inherit selected native assistant: %v", err)
	}
	if err := e.Accept(context.Background(), task.ID); err != nil {
		t.Fatalf("future selection invalidated acceptance: %v", err)
	}
	message, err := e.startLinkedBatch(task.ID, messageRequest{ID: "assistant-next-batch", Kind: messageRequirement, Content: "Continue this project with the selected native assistant"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := e.Store.Get(message.LinkedTask)
	if err != nil || child.ModelPlan != changed || child.ParentTaskID != task.ID || len(child.PreviousContext) == 0 {
		t.Fatalf("linked work lost the selected assistant or parent context: %v", err)
	}
}

type assistantResumeAgent struct {
	base       domainFixture
	failDomain string
	remaining  int
	seen       []RoleRequest
}

func (agent *assistantResumeAgent) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	agent.seen = append(agent.seen, request)
	if request.Role == roleImplementer && request.Domain == agent.failDomain && agent.remaining > 0 {
		agent.remaining--
		return nil, context.DeadlineExceeded
	}
	return agent.base.Run(ctx, request)
}

func TestAssistantSwitchResumesFailedNativeStageWithinTheSameRound(t *testing.T) {
	for _, domain := range []string{domainFrontend, domainBackend} {
		t.Run(domain, func(t *testing.T) {
			base := &domainFixture{}
			e, task := domainEngine(t, base, domainCheck)
			current, err := e.Store.Get(task.ID)
			if err != nil {
				t.Fatal(err)
			}
			initial := current.ModelPlan
			initial.AssistantMode, initial.Auditor, initial.Frontend, initial.Backend = connectionHarness, connectionHarness, connectionHarness, connectionHarness
			initial.AuditorModel, initial.FrontendModel, initial.BackendModel = "", "", ""
			if _, err := e.UpdateModelPlan(task.ID, 1, initial); err != nil {
				t.Fatal(err)
			}
			agent := &assistantResumeAgent{failDomain: domain, remaining: 3}
			e.Runner = agent
			if err := e.RunTask(context.Background(), task.ID); err == nil {
				t.Fatal("native operational failure did not pause")
			}
			paused, _ := e.Store.Get(task.ID)
			if paused.Round != 1 || !domainStageResumeEligible(paused) {
				t.Fatal("failure spent quality rounds or lost same-round recovery")
			}
			changed := paused.ModelPlan
			changed.AssistantMode, changed.Auditor, changed.Frontend, changed.Backend = connectionCodex, connectionCodex, connectionCodex, connectionCodex
			changed.FrontendModel, changed.BackendModel = "", ""
			if _, err := e.UpdateModelPlan(task.ID, paused.PlanRevision, changed); err != nil {
				t.Fatal(err)
			}
			if err := e.RunTask(context.Background(), task.ID); err != nil {
				t.Fatal(err)
			}
			saved, err := e.Store.Get(task.ID)
			if err != nil || saved.State != stateAwaitingAcceptance || saved.Round != 1 || saved.Worktree != paused.Worktree || agent.base.frontend != 1 || agent.base.backend != 1 {
				t.Fatalf("switch repeated completed work or reset progress: %v", err)
			}
			for _, run := range saved.Runs[len(paused.Runs):] {
				if run.ProviderID != connectionCodex {
					t.Fatal("pending stage ignored the saved assistant switch")
				}
			}
			if domain == domainBackend && saved.Runs[1].ProviderID != connectionHarness {
				t.Fatal("completed frontend history was rewritten")
			}
			if err := e.Accept(context.Background(), task.ID); err != nil {
				t.Fatalf("native switch lost valid evidence: %v", err)
			}
		})
	}
}
