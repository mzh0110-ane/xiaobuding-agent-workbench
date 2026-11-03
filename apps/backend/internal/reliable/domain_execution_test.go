package reliable

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type domainFixture struct {
	total, frontend, backend, reviews int
	violate                           bool
	checkPrompt                       bool
}

func (a *domainFixture) Run(_ context.Context, req RoleRequest) (*AgentResult, error) {
	a.total++
	var raw string
	switch req.Role {
	case roleAuditor:
		raw = `{"summary":"Inspect both domains","facts":["frontend.txt and backend.txt exist"],"inferences":[],"unknowns":[],"read_files":["frontend.txt","backend.txt"],"required_read_files":["frontend.txt","backend.txt"],"allowed_paths":["frontend.txt","backend.txt"],"frontend_paths":["frontend.txt"],"backend_paths":["backend.txt"],"plan":["Frontend updates its file, then backend consumes that output"]}`
	case roleImplementer:
		encoded, _ := json.Marshal(req)
		var fields map[string]any
		_ = json.Unmarshal(encoded, &fields)
		domain, _ := fields["Domain"].(string)
		path, content := "frontend.txt", "front-complete\n"
		if domain == domainBackend {
			if a.checkPrompt && strings.Contains(req.Prompt, `"files":`) {
				return nil, fmt.Errorf("backend prompt embedded whole source manifests")
			}
			a.backend++
			front, _ := os.ReadFile(filepath.Join(req.Directory, "frontend.txt"))
			if string(front) != content {
				return nil, fmt.Errorf("backend did not receive the frontend output: %q", front)
			}
			path, content = "backend.txt", "back-complete\n"
		} else {
			a.frontend++
		}
		if a.violate {
			path = "backend.txt"
		}
		if err := os.WriteFile(filepath.Join(req.Directory, path), []byte(content), 0600); err != nil {
			return nil, err
		}
		raw = `{"summary":"Domain updated","read_files":["frontend.txt","backend.txt"],"docs_impact":"No docs change","residual_risks":[]}`
	case roleReviewer:
		a.reviews++
		raw = `{"verdict":"pass","summary":"Both domains and checks reviewed","read_files":["frontend.txt","backend.txt"],"findings":[]}`
	}
	return &AgentResult{Session: fmt.Sprintf("domain-session-%d", a.total), Output: json.RawMessage(raw)}, nil
}

func TestBackendHandoffUsesEvidenceReferencesInsteadOfWholeManifests(t *testing.T) {
	e, task := domainEngine(t, &domainFixture{checkPrompt: true}, domainCheck)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
}

func domainEngine(t *testing.T, agent *domainFixture, command string) (*Engine, *Task) {
	t.Helper()
	e, task := fixtureEngine(t, agent, command)
	e.Providers = testProviders()
	for _, name := range []string{"frontend.txt", "backend.txt"} {
		if err := os.WriteFile(filepath.Join(task.Project, name), []byte("before\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := git(context.Background(), task.Project, "add", "frontend.txt", "backend.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := git(context.Background(), task.Project, "-c", "user.name=Fixture", "-c", "user.email=fixture@local.test", "commit", "-m", "Domain fixture"); err != nil {
		t.Fatal(err)
	}
	var plan ModelPlan
	_ = json.Unmarshal([]byte(`{"mode":"fullstack","auditor":"codex","implementer":"codex","reviewer":"codex","frontend":"deepseek","backend":"codex","frontend_model":"deepseek-fixture","backend_model":"codex-fixture"}`), &plan)
	if _, err := e.UpdateModelPlan(task.ID, 0, plan); err != nil {
		t.Fatal(err)
	}
	return e, task
}

const domainCheck = `python -c "from pathlib import Path; assert Path('frontend.txt').read_text().strip() == 'front-complete'; assert Path('backend.txt').read_text().strip() == 'back-complete'"`

func TestBothDomainsShareOneRoundAndVerifiedWorktree(t *testing.T) {
	agent := &domainFixture{}
	e, task := domainEngine(t, agent, domainCheck)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(task.ID)
	if got.State != stateAwaitingAcceptance || got.Round != 1 || agent.frontend != 1 || agent.backend != 1 || len(got.Runs) != 4 {
		t.Fatalf("domain work was not one complete shared round: %+v", got)
	}
	if got.Runs[1].ProviderID != "deepseek" || got.Runs[2].ProviderID != "codex" {
		t.Fatal("frontend/backend responsibilities were not routed independently")
	}
	if err := e.Accept(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
}

func TestFrontendScopeViolationStopsBeforeBackendAndReview(t *testing.T) {
	agent := &domainFixture{violate: true}
	e, task := domainEngine(t, agent, domainCheck)
	if err := e.RunTask(context.Background(), task.ID); err == nil {
		t.Fatal("frontend changed a backend file without blocking")
	}
	got, _ := e.Store.Get(task.ID)
	if got.State != stateBlocked || agent.frontend != 1 || agent.backend != 0 || agent.reviews != 0 {
		t.Fatalf("scope violation continued into later work: %+v %+v", got, agent)
	}
}

func TestDualDomainFailureStillExhaustsThreeTotalRounds(t *testing.T) {
	agent := &domainFixture{}
	e, task := domainEngine(t, agent, `python -c "raise SystemExit(1)"`)
	_ = e.RunTask(context.Background(), task.ID)
	got, _ := e.Store.Get(task.ID)
	if got.State != stateExhausted || got.Round != 3 || agent.frontend != 3 || agent.backend != 3 || agent.reviews != 0 {
		t.Fatalf("domain splitting changed the total budget: state=%s round=%d agent=%+v", got.State, got.Round, agent)
	}
}
