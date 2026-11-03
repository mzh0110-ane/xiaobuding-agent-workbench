package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type pausedDomain struct {
	fixture domainFixture
	started chan RoleRequest
	release chan struct{}
}

func (a *pausedDomain) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	if req.Domain == domainFrontend {
		a.started <- req
		select {
		case <-a.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return a.fixture.Run(ctx, req)
}

func TestDomainModelSwitchFreezesCurrentAndUpdatesNextWriter(t *testing.T) {
	e, task := domainEngine(t, &domainFixture{}, domainCheck)
	for i := range e.Providers {
		p := &e.Providers[i]
		p.Models = []ModelOption{{ID: p.Model, DefaultEffort: "low", Efforts: []string{"low", "high"}}, {ID: p.Model + "-alternate", DefaultEffort: "low", Efforts: []string{"low", "high"}}}
	}
	agent := &pausedDomain{started: make(chan RoleRequest, 1), release: make(chan struct{})}
	e.Runner = agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() { done <- e.RunTask(ctx, task.ID) }()
	released := false
	finished := false
	t.Cleanup(func() {
		cancel()
		if !released {
			close(agent.release)
		}
		if !finished {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("domain worker did not stop")
			}
		}
	})
	current := <-agent.started
	saved, _ := e.Store.Get(task.ID)
	plan := saved.ModelPlan
	plan.BackendModel = "codex-fixture-alternate"
	plan.BackendEffort = "high"
	plan.FrontendModel = "deepseek-fixture-alternate"
	if _, err := e.UpdateModelPlan(task.ID, 1, plan); err != nil {
		t.Fatal(err)
	}
	close(agent.release)
	released = true
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	finished = true
	got, _ := e.Store.Get(task.ID)
	if current.Model != "deepseek-fixture" || current.Effort != "low" || got.Runs[1].PlanRevision != 1 || got.Runs[2].Model != "codex-fixture-alternate" || got.Runs[2].Effort != "high" || got.Round != 1 {
		t.Fatalf("switch lost frozen/native identity: current=%+v runs=%+v", current, got.Runs)
	}
}

func TestMissingDomainEvidenceRejectsAcceptance(t *testing.T) {
	e, task := domainEngine(t, &domainFixture{}, domainCheck)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(task.ID)
	if len(got.DomainEvidence) != 2 {
		t.Fatalf("missing per-domain evidence: %v", got.DomainEvidence)
	}
	if err := os.Remove(filepath.Join(e.Root, "evidence", task.ID, got.DomainEvidence[0])); err != nil {
		t.Fatal(err)
	}
	if err := e.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("missing frontend evidence was accepted")
	}
}

func TestDomainAndScopeCannotBeChangedAfterTaskCreation(t *testing.T) {
	e, task := domainEngine(t, &domainFixture{}, domainCheck)
	saved, _ := e.Store.Get(task.ID)
	for _, kind := range []string{"mode", "scope"} {
		plan := saved.ModelPlan
		if kind == "mode" {
			plan.Mode = "frontend"
		} else {
			plan.FrontendPaths = "frontend.txt"
		}
		if _, err := e.UpdateModelPlan(task.ID, 1, plan); err == nil {
			t.Fatalf("running contract admitted %s change", kind)
		}
	}
}

func TestLegacyDispatchSerializationOmitsNewFields(t *testing.T) {
	d := Dispatch{ID: "old", Role: "implementer", Round: 1, ProviderID: "codex", ProviderName: "Codex", Model: "old-model"}
	raw, _ := json.Marshal(d)
	want := `{"id":"old","role":"implementer","round":1,"provider_id":"codex","provider_name":"Codex","model":"old-model","plan_revision":0,"source_hash":"","session":"","status":"","started":"","completed":"","input_evidence":"","start_evidence":"","end_evidence":""}`
	if string(raw) != want {
		t.Fatalf("new empty fields invalidate legacy dispatch hashes: %s", raw)
	}
}
