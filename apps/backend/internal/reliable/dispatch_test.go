package reliable

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type gatedAgent struct {
	fixture fixtureAgent
	started chan RoleRequest
	release chan struct{}
	seen    []RoleRequest
}

func (a *gatedAgent) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	a.seen = append(a.seen, req)
	if req.Role == roleAuditor {
		a.started <- req
		select {
		case <-a.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return a.fixture.Run(ctx, req)
}

func testProviders() []Provider {
	return []Provider{
		{ID: "deepseek", Name: "DeepSeek", Model: "deepseek-fixture", Available: true},
		{ID: "codex", Name: "Codex", Model: "codex-fixture", Available: true},
	}
}

func TestRunningModelSwitchPreservesCurrentCallAndSharedProgress(t *testing.T) {
	agent := &gatedAgent{started: make(chan RoleRequest, 1), release: make(chan struct{})}
	e, task := fixtureEngine(t, agent, passingCheck)
	e.Providers = testProviders()
	initial := ModelPlan{Auditor: "deepseek", Implementer: "deepseek", Reviewer: "deepseek"}
	if _, err := e.UpdateModelPlan(task.ID, 0, initial); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan struct{})
	result := make(chan error, 1)
	var release sync.Once
	go func() { defer close(done); result <- e.RunTask(ctx, task.ID) }()
	t.Cleanup(func() { cancel(); release.Do(func() { close(agent.release) }); <-done })
	var active RoleRequest
	select {
	case active = <-agent.started:
	case <-ctx.Done():
		t.Fatal("audit did not start")
	}
	if active.ProviderID != "deepseek" {
		t.Fatalf("active audit did not freeze the planned provider: %q", active.ProviderID)
	}
	changed := ModelPlan{Auditor: "codex", Implementer: "codex", Reviewer: "deepseek"}
	if _, err := e.UpdateModelPlan(task.ID, 1, changed); err != nil {
		t.Fatal(err)
	}
	release.Do(func() { close(agent.release) })
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	assertMixedDispatch(t, e, task.ID, agent.seen)
}

func assertMixedDispatch(t *testing.T, e *Engine, id string, seen []RoleRequest) {
	t.Helper()
	got, err := e.Store.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != stateAwaitingAcceptance || got.Round != 1 || len(got.Runs) != 3 || len(seen) != 3 {
		t.Fatalf("switch lost shared task progress: %+v", got)
	}
	want := []string{"deepseek", "codex", "deepseek"}
	for i, provider := range want {
		if seen[i].ProviderID != provider || got.Runs[i].ProviderID != provider || got.Runs[i].Session == "" || got.Runs[i].SourceHash == "" {
			t.Fatalf("dispatch %d did not use its actual frozen identity: %+v", i, got.Runs[i])
		}
	}
	if got.Runs[0].PlanRevision != 1 || got.Runs[1].PlanRevision != 2 || got.PlanRevision != 2 {
		t.Fatal("a later selection rewrote the active call or was ignored by its successor")
	}
	if err = e.Accept(context.Background(), id); err != nil {
		t.Fatalf("mixed-provider evidence could not be accepted: %v", err)
	}
}

func TestDispatchHandoffEvidenceCannotDisappearBeforeAcceptance(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(task.ID)
	if len(got.Runs) != 3 {
		t.Fatalf("actual stage execution has no durable handoff history: %d", len(got.Runs))
	}
	path := filepath.Join(e.Root, "evidence", task.ID, got.Runs[1].InputEvidence)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := e.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("candidate with missing model handoff evidence was applied")
	}
}
