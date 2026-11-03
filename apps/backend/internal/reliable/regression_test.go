package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type agentFunc func(context.Context, RoleRequest) (*AgentResult, error)

func (f agentFunc) Run(ctx context.Context, r RoleRequest) (*AgentResult, error) { return f(ctx, r) }

func alterReport(result *AgentResult, key string, value any) {
	var report map[string]any
	_ = json.Unmarshal(result.Output, &report)
	report[key] = value
	result.Output, _ = json.Marshal(report)
}

func TestAuxiliaryAuditReadsAreNotMandatoryImplementationReads(t *testing.T) {
	base := &fixtureAgent{}
	agent := agentFunc(func(ctx context.Context, r RoleRequest) (*AgentResult, error) {
		result, err := base.Run(ctx, r)
		if err != nil {
			return nil, err
		}
		if r.Role == "auditor" {
			alterReport(result, "read_files", []string{"app.txt", ".gitignore"})
			alterReport(result, "required_read_files", []string{"app.txt"})
		}
		return result, nil
	})
	e, task := fixtureEngine(t, agent, passingCheck)
	if err := os.WriteFile(filepath.Join(task.Project, ".gitignore"), []byte("__pycache__/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, task.Project, "add", ".")
	gitTest(t, task.Project, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "add ignore rules")
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(task.ID)
	if got.State != "awaiting_acceptance" {
		t.Fatalf("auxiliary read blocked implementation: %s", got.State)
	}
}

func TestImplementationCanReportReadingAFileItThenDeletes(t *testing.T) {
	base := &fixtureAgent{}
	agent := agentFunc(func(ctx context.Context, r RoleRequest) (*AgentResult, error) {
		result, err := base.Run(ctx, r)
		if err == nil && r.Role == "implementer" {
			err = os.Remove(filepath.Join(r.Directory, "app.txt"))
		}
		return result, err
	})
	e, task := fixtureEngine(t, agent, `python -c "from pathlib import Path; assert not Path('app.txt').exists()"`)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(task.ID)
	if got.State != "awaiting_acceptance" {
		t.Fatalf("legitimate deletion blocked: %s", got.State)
	}
}

func TestReworkPreservesEarlierAuthorizedChanges(t *testing.T) {
	base := &fixtureAgent{rejectFirst: true}
	agent := agentFunc(func(ctx context.Context, r RoleRequest) (*AgentResult, error) {
		result, err := base.Run(ctx, r)
		if err != nil {
			return nil, err
		}
		if r.Role == "auditor" && base.implement == 0 {
			alterReport(result, "allowed_paths", []string{"app.txt", "notes.txt"})
		}
		if r.Role == "implementer" && base.implement == 1 {
			err = os.WriteFile(filepath.Join(r.Directory, "notes.txt"), []byte("authorized earlier\n"), 0600)
		}
		if r.Role == "reviewer" {
			alterReport(result, "read_files", []string{"app.txt", "notes.txt"})
		}
		return result, err
	})
	e, task := fixtureEngine(t, agent, passingCheck)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(task.ID)
	if got.State != "awaiting_acceptance" || got.Round != 2 {
		t.Fatalf("earlier authorization was lost: %+v", got)
	}
}

func TestReviewCannotInventAdditionalSourceReads(t *testing.T) {
	base := &fixtureAgent{}
	agent := agentFunc(func(ctx context.Context, r RoleRequest) (*AgentResult, error) {
		result, err := base.Run(ctx, r)
		if err == nil && r.Role == "reviewer" {
			alterReport(result, "read_files", []string{"app.txt", "imaginary.py"})
		}
		return result, err
	})
	e, task := fixtureEngine(t, agent, passingCheck)
	_ = e.RunTask(context.Background(), task.ID)
	got, _ := e.Store.Get(task.ID)
	if got.State != "blocked" {
		t.Fatalf("invented evidence accepted: %s", got.State)
	}
}

func TestWorktreeCreationRecoversTheSameInitialCheckout(t *testing.T) {
	project := fixtureRepo(t)
	root := t.TempDir()
	ctx := context.Background()
	first, base, err := PrepareWorktree(ctx, root, project, "recover")
	if err != nil {
		t.Fatal(err)
	}
	second, recoveredBase, err := PrepareWorktree(ctx, root, project, "recover")
	if err != nil {
		t.Fatal(err)
	}
	if first != second || base != recoveredBase {
		t.Fatal("recovery changed workspace identity")
	}
}
