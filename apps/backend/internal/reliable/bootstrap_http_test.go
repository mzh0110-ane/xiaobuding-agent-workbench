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

type bootstrapAgent struct {
	calls, writes, reviews int
	noChecks               bool
}

func (a *bootstrapAgent) Run(_ context.Context, r RoleRequest) (*AgentResult, error) {
	a.calls++
	var report any
	switch r.Role {
	case roleAuditor:
		reads := []string{}
		if _, err := os.Stat(filepath.Join(r.Directory, "app.cjs")); err == nil {
			reads = []string{"app.cjs"}
		}
		report = Audit{Summary: "Inspect current folder and create a tested function", Facts: []string{"Current folder inspected"}, ReadFiles: reads, RequiredReadFiles: reads, AllowedPaths: []string{"."}, Plan: []string{"Implement the requested function and real assertions"}}
	case roleImplementer:
		a.writes++
		files := map[string]string{"app.cjs": "module.exports = () => 42;\n"}
		if !a.noChecks {
			files["package.json"] = `{"scripts":{"test":"node --test app.test.cjs"}}`
			files["app.test.cjs"] = "const assert = require('node:assert/strict'); const test = require('node:test'); test('returns 42', () => assert.equal(require('./app.cjs')(), 42));\n"
		}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(r.Directory, name), []byte(content), 0600); err != nil {
				return nil, err
			}
		}
		report = implementation{Summary: "Created the function and checks", ReadFiles: []string{"app.cjs"}, DocsImpact: "No existing public documentation"}
	case roleReviewer:
		a.reviews++
		report = map[string]any{"verdict": "pass", "summary": "Read implementation, real test assertion and raw execution logs", "read_files": []string{"app.cjs", "app.test.cjs", "package.json"}, "findings": []any{}}
	}
	raw, err := json.Marshal(report)
	return &AgentResult{Session: fmt.Sprintf("bootstrap-fresh-%d", a.calls), Output: raw}, err
}

func TestEmptyFolderCreatesRealChecksWithoutManualConfiguration(t *testing.T) {
	agent := &bootstrapAgent{}
	h := localFolderHTTP(t, agent)
	project := t.TempDir()
	task := createLocalFolderTask(t, h, project, nil)
	got := awaitLocalFolderTask(t, h, task.ID)
	if got.State != stateAwaitingAcceptance || got.Round != 1 || len(got.Checks) != 1 || got.Checks[0].Command != "npm run test" || got.Checks[0].ExitCode != 0 {
		t.Fatalf("empty folder did not produce real checked evidence: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(project, "app.cjs")); !os.IsNotExist(err) {
		t.Fatal("new source reached the original before acceptance")
	}
	if agent.reviews != 1 {
		t.Fatal("new project bypassed independent review")
	}
	if err := h.Engine.Accept(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(filepath.Join(project, "app.cjs")); err != nil || len(raw) == 0 {
		t.Fatal("accepted source was not applied")
	}
}

func TestMissingAutomaticChecksUsesThreeRoundsAndCannotBeAccepted(t *testing.T) {
	agent := &bootstrapAgent{noChecks: true}
	h := localFolderHTTP(t, agent)
	task := createLocalFolderTask(t, h, t.TempDir(), nil)
	got := awaitLocalFolderTask(t, h, task.ID)
	if got.State != stateExhausted || got.Round != 3 || agent.writes != 3 || agent.reviews != 0 {
		t.Fatalf("missing checks were not bounded fail-closed: state=%s round=%d writes=%d reviews=%d reason=%s", got.State, got.Round, agent.writes, agent.reviews, got.Reason)
	}
	if err := h.Engine.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("a project with no actual checks was accepted")
	}
}

func TestAutomaticCheckPlanTamperingBlocksAcceptance(t *testing.T) {
	h := localFolderHTTP(t, &bootstrapAgent{})
	task := createLocalFolderTask(t, h, t.TempDir(), nil)
	got := awaitLocalFolderTask(t, h, task.ID)
	if got.State != stateAwaitingAcceptance || got.CheckPlanEvidence == "" {
		t.Fatalf("no frozen automatic plan: %s %s", got.State, got.Reason)
	}
	path := filepath.Join(h.Engine.Root, "evidence", got.ID, got.CheckPlanEvidence)
	if err := os.WriteFile(path, []byte(`{"commands":["echo fake"],"source_hash":"fake","round":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := h.Engine.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("a modified automatic check plan was accepted")
	}
}

func TestEmptyAuditExceptionEndsAsSoonAsSourceExists(t *testing.T) {
	task := &Task{AutoChecks: true, AllowedPaths: []string{"."}}
	audit := &Audit{Summary: "Plan", AllowedPaths: []string{"."}, Plan: []string{"Implement"}}
	if err := validateTaskAudit(task, audit, &Snapshot{Files: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if err := validateTaskAudit(task, audit, &Snapshot{Files: map[string]string{"app.cjs": "fingerprint"}}); err == nil {
		t.Fatal("later rework skipped reading existing source")
	}
}

func TestAutomaticChecksDoNotTreatZeroTestRunnersAsVerification(t *testing.T) {
	for command, output := range map[string]string{"python -m unittest": "Ran 0 tests in 0.000s\n\nOK\n", "npm run test": "# tests 0\n# pass 0\n", "go test ./...": "? module/example [no test files]\n"} {
		path := filepath.Join(t.TempDir(), "check.log")
		if err := os.WriteFile(path, []byte(output), 0600); err != nil {
			t.Fatal(err)
		}
		if !automaticCheckHasNoTests(command, path) {
			t.Fatalf("zero tests reported as verification: %q", output)
		}
	}
}

func TestAutomaticGoChecksAllowPackagesWithoutTestsAlongsideRealTests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check.log")
	if err := os.WriteFile(path, []byte("? module/cmd [no test files]\nok module/business 0.123s\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if automaticCheckHasNoTests("go test ./...", path) {
		t.Fatal("a successful real test package was rejected because another package has no tests")
	}
}

func TestZeroTestExecutionRemainsUnverifiedWithoutChangingActualExit(t *testing.T) {
	h := localFolderHTTP(t, &fixtureAgent{})
	dir := t.TempDir()
	task := &Task{Worktree: dir, AutoChecks: true, Commands: []string{`python -c "print('Ran 0 tests in 0.000s')"`}}
	checks, passed, err := h.Engine.executeVerification(context.Background(), task, dir)
	if err != nil || passed || len(checks) != 1 || checks[0].ExitCode != 0 {
		t.Fatalf("zero-test execution must retain exit 0 without passing verification: %+v %v", checks, err)
	}
	raw, err := json.Marshal(checks)
	if err != nil || !strings.Contains(string(raw), "verification_problem") {
		t.Fatalf("exit-only UI cannot distinguish zero tests from verified behavior: %s %v", raw, err)
	}
}
