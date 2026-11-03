package reliable

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func scopeAudit() Audit {
	return Audit{Summary: "Inspect both domains", Plan: []string{"Implement separately"},
		AllowedPaths:  []string{"frontend/", "backend/"},
		FrontendPaths: []string{"frontend/"}, BackendPaths: []string{"backend/"}}
}

func TestAuditDirectoryScopesAreCanonicalAcrossValidationAndDispatch(t *testing.T) {
	agent := &directoryScopeAgent{}
	e, task := domainEngine(t, &agent.fixture, domainCheck)
	e.Runner = agent
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, err := e.Store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != stateAwaitingAcceptance || got.Round != 1 || agent.fixture.frontend != 1 || agent.fixture.backend != 1 {
		t.Fatalf("directory notation prevented a verified shared round: state=%s round=%d", got.State, got.Round)
	}
	if !containsPath(got.Audit.AllowedPaths, "artifacts/result") || !containsPath(got.Audit.FrontendPaths, "artifacts/result") {
		t.Fatalf("scope was not canonical before persistence and handoff: %+v", got.Audit)
	}
}

type directoryScopeAgent struct{ fixture domainFixture }

func (a *directoryScopeAgent) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	result, err := a.fixture.Run(ctx, req)
	if err != nil || req.Role != roleAuditor {
		return result, err
	}
	var audit Audit
	if err := json.Unmarshal(result.Output, &audit); err != nil {
		return nil, err
	}
	audit.AllowedPaths = append(audit.AllowedPaths, "artifacts/result/")
	audit.FrontendPaths = append(audit.FrontendPaths, "artifacts/result/")
	result.Output, err = json.Marshal(audit)
	return result, err
}

func TestAuditDirectoryScopesStillRejectInvalidPathsWithLocation(t *testing.T) {
	paths := []string{"../outside/", "frontend/../backend/", "/absolute/", "C:/outside/", "frontend\\nested/", ".git/", "frontend//", "/", ""}
	for _, field := range []string{"allowed_paths", "frontend_paths", "backend_paths"} {
		for _, path := range paths {
			t.Run(field+"/"+path, func(t *testing.T) {
				audit := scopeAudit()
				switch field {
				case "allowed_paths":
					audit.AllowedPaths = []string{path}
				case "frontend_paths":
					audit.FrontendPaths = []string{path}
				case "backend_paths":
					audit.BackendPaths = []string{path}
				}
				err := validateAuditPlan(&audit, []string{"."})
				if err == nil || !strings.Contains(err.Error(), field+"[0]") {
					t.Fatalf("invalid scope must identify its field and entry: path=%q error=%v", path, err)
				}
			})
		}
	}
}

func TestAuditDirectoryScopesPreserveAuthorizationAndDomainSeparation(t *testing.T) {
	audit := scopeAudit()
	if err := validateAuditPlan(&audit, []string{"frontend", "backend"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(audit.AllowedPaths, []string{"frontend", "backend"}) {
		t.Fatalf("unexpected canonical scope: %v", audit.AllowedPaths)
	}
	plan := ModelPlan{Mode: modeFullstack}
	audit.BackendPaths = []string{"frontend/child/"}
	if err := validateDomainAudit(plan, &audit); err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("directory aliases must not bypass domain separation: %v", err)
	}
	audit = scopeAudit()
	if err := validateAuditPlan(&audit, []string{"frontend"}); err == nil || !strings.Contains(err.Error(), "user-authorized") {
		t.Fatalf("canonicalization must not broaden user authorization: %v", err)
	}
}
