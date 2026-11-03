package reliable

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type fixtureAgent struct {
	implement, review, total  int
	rejectFirst, mutateReview bool
}

func (a *fixtureAgent) Run(_ context.Context, r RoleRequest) (*AgentResult, error) {
	a.total++
	var raw string
	switch r.Role {
	case "auditor":
		raw = `{"summary":"Read source","facts":["app.txt exists"],"inferences":[],"unknowns":[],"read_files":["app.txt"],"required_read_files":["app.txt"],"allowed_paths":["app.txt"],"plan":["Update app.txt"]}`
	case "implementer":
		a.implement++
		if err := os.WriteFile(filepath.Join(r.Directory, "app.txt"), []byte("reviewed\n"), 0600); err != nil {
			return nil, err
		}
		raw = `{"summary":"Updated source","read_files":["app.txt"],"docs_impact":"No public documentation impact","residual_risks":[]}`
	case "reviewer":
		a.review++
		if a.mutateReview {
			if err := os.WriteFile(filepath.Join(r.Directory, "app.txt"), []byte("unreviewed\n"), 0600); err != nil {
				return nil, err
			}
		}
		if a.rejectFirst && a.review == 1 {
			raw = `{"verdict":"rework","summary":"Missing required behavior","read_files":["app.txt"],"findings":[{"severity":"P1","file":"app.txt","reason":"Incomplete implementation"}]}`
		} else {
			raw = `{"verdict":"pass","summary":"Read actual source and verification","read_files":["app.txt"],"findings":[]}`
		}
	}
	return &AgentResult{Session: fmt.Sprintf("fresh-%d", a.total), Output: json.RawMessage(raw)}, nil
}

func fixtureEngine(t *testing.T, agent AgentRunner, command string) (*Engine, *Task) {
	t.Helper()
	root := t.TempDir()
	store, err := OpenStore(filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	task := &Task{ID: "task", Project: fixtureRepo(t), Goal: "Update app.txt", Commands: []string{command}, State: "queued"}
	if err := store.Create(task); err != nil {
		t.Fatal(err)
	}
	return &Engine{Store: store, Root: root, Runner: agent}, task
}

const passingCheck = `python -c "from pathlib import Path; assert Path('app.txt').read_text().strip() == 'reviewed'"`

func TestFailedVerificationStopsAfterThreeRounds(t *testing.T) {
	agent := &fixtureAgent{}
	e, task := fixtureEngine(t, agent, `python -c "raise SystemExit(1)"`)
	_ = e.RunTask(context.Background(), task.ID)
	got, err := e.Store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "exhausted" || got.Round != 3 || agent.implement != 3 {
		t.Fatalf("not bounded: state=%s round=%d calls=%d", got.State, got.Round, agent.implement)
	}
	if err := e.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("failed candidate was accepted")
	}
	if err := e.RunTask(context.Background(), task.ID); err == nil {
		t.Fatal("exhausted task restarted")
	}
}

func TestIndependentReviewRequestsReworkThenHumanAcceptance(t *testing.T) {
	agent := &fixtureAgent{rejectFirst: true}
	e, task := fixtureEngine(t, agent, passingCheck)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(task.ID)
	if got.State != "awaiting_acceptance" || got.Round != 2 || agent.review != 2 {
		t.Fatalf("unexpected result: %+v", got)
	}
	before, _ := os.ReadFile(filepath.Join(task.Project, "app.txt"))
	if string(before) != "before\n" {
		t.Fatal("original changed before acceptance")
	}
	if err := e.Accept(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(task.Project, "app.txt"))
	if string(after) != "reviewed\n" {
		t.Fatal("human acceptance did not apply reviewed result")
	}
}

func TestReviewerCannotChangeCandidateAndPass(t *testing.T) {
	agent := &fixtureAgent{mutateReview: true}
	e, task := fixtureEngine(t, agent, passingCheck)
	_ = e.RunTask(context.Background(), task.ID)
	got, _ := e.Store.Get(task.ID)
	if got.State != "blocked" {
		t.Fatalf("review mutation passed: %s", got.State)
	}
	if err := e.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("mutated candidate accepted")
	}
}

func TestSourceDriftInvalidatesHumanAcceptance(t *testing.T) {
	agent := &fixtureAgent{}
	e, task := fixtureEngine(t, agent, passingCheck)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(task.ID)
	if err := os.WriteFile(filepath.Join(got.Worktree, "app.txt"), []byte("changed after review"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := e.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("stale review accepted")
	}
	raw, _ := os.ReadFile(filepath.Join(task.Project, "app.txt"))
	if string(raw) != "before\n" {
		t.Fatal("stale result reached original")
	}
}
