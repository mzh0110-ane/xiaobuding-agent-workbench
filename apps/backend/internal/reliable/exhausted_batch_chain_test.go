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

type inheritedFilesAgent struct{ fixtureAgent }

func (a *inheritedFilesAgent) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	var raw string
	switch request.Role {
	case roleAuditor:
		got, err := os.ReadFile(filepath.Join(request.Directory, "inherited.txt"))
		if err != nil || string(got) != "Retained from the previous batch\n" {
			return nil, fmt.Errorf("missing inherited file before fresh audit: %s %v", got, err)
		}
		raw = `{"summary":"Read inherited source","facts":["app.txt and inherited.txt were read"],"inferences":[],"unknowns":[],"read_files":["app.txt","inherited.txt"],"required_read_files":["app.txt","inherited.txt"],"allowed_paths":["app.txt"],"plan":["Update app.txt and retain inherited.txt"]}`
	case roleImplementer:
		a.total++
		a.implement++
		if err := os.WriteFile(filepath.Join(request.Directory, "app.txt"), []byte("reviewed\n"), 0600); err != nil {
			return nil, err
		}
		raw = `{"summary":"Updated source and retained inherited file","read_files":["app.txt","inherited.txt"],"docs_impact":"No public documentation impact","residual_risks":[]}`
	case roleReviewer:
		a.review++
		raw = `{"verdict":"pass","summary":"Read cumulative changes and actual verification","read_files":["app.txt","inherited.txt"],"findings":[]}`
	default:
		return a.fixtureAgent.Run(ctx, request)
	}
	a.total++
	return &AgentResult{Session: fmt.Sprintf("new-batch-fresh-%d", a.total), Output: json.RawMessage(raw)}, nil
}

// @covers AC-RELIABLE-RENEWAL-001.2
// @covers AC-RELIABLE-RENEWAL-001.5
func TestExhaustedBatchAcceptanceAppliesCumulativeCandidate(t *testing.T) {
	agent := &inheritedFilesAgent{}
	e, parent := preparedExhaustedTask(t, agent)
	parent, err := e.Store.Update(parent.ID, func(saved *Task) error {
		saved.ApprovedPaths = []string{"app.txt", "inherited.txt"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained partial app\n")
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "inherited.txt"), "Retained from the previous batch\n")
	oldIndex := gitTest(t, parent.Worktree, "ls-files", "--stage")
	child := claimExhaustedChild(t, e, parent, "cumulative")
	if err = e.RunTask(context.Background(), child.ID); err != nil {
		t.Fatal(err)
	}
	child, _ = e.Store.Get(child.ID)
	if child.State != stateAwaitingAcceptance || child.Round != 1 {
		t.Fatalf("candidate failed fresh verification: state=%s reason=%s", child.State, child.Reason)
	}
	before, _ := os.ReadFile(filepath.Join(parent.Project, "app.txt"))
	if string(before) != "before\n" {
		t.Fatal("candidate changed the original before acceptance")
	}
	if _, err = os.Stat(filepath.Join(parent.Project, "inherited.txt")); !os.IsNotExist(err) {
		t.Fatal("inherited file reached the original without acceptance")
	}
	if err = e.Accept(context.Background(), child.ID); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct{ name, want string }{{"app.txt", "reviewed\n"}, {"inherited.txt", "Retained from the previous batch\n"}} {
		got, readErr := os.ReadFile(filepath.Join(parent.Project, check.name))
		if readErr != nil || string(got) != check.want {
			t.Fatalf("acceptance lost cumulative source %s: %s %v", check.name, got, readErr)
		}
	}
	if oldIndex != gitTest(t, parent.Worktree, "ls-files", "--stage") {
		t.Fatal("continuation changed the previous candidate index")
	}
	oldApp, _ := os.ReadFile(filepath.Join(parent.Worktree, "app.txt"))
	if string(oldApp) != "retained partial app\n" {
		t.Fatal("continuation changed the previous candidate source")
	}
}

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedBatchPreservesMeaningfulAutomaticVerification(t *testing.T) {
	e, parent := preparedExhaustedTask(t, &fixtureAgent{})
	parent, err := e.Store.Update(parent.ID, func(saved *Task) error {
		saved.AutoChecks, saved.Commands = true, nil
		saved.DiscoveredCommands = []string{"npm run test"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	child := claimExhaustedChild(t, e, parent, "checks")
	child.Worktree = fixtureRepo(t)
	child.Round = 1
	mustRecoveryWrite(t, filepath.Join(child.Worktree, "package.json"), `{"scripts":{"test":"python -c \"print('Ran 0 tests in 0.000s')\"","typecheck":"python -c \"print('fresh-typecheck-ran')\""}}`)
	if _, err = e.Store.Update(child.ID, func(saved *Task) error { saved.Worktree, saved.Round = child.Worktree, child.Round; return nil }); err != nil {
		t.Fatal(err)
	}
	snap, err := TakeSnapshot(context.Background(), child.Worktree)
	if err != nil {
		t.Fatal(err)
	}
	if reason, setupErr := e.ensureAutomaticChecks(child, snap); setupErr != nil || reason != "" {
		t.Fatalf("fresh automatic plan did not freeze: %s %v", reason, setupErr)
	}
	logs := t.TempDir()
	checks, passed, err := e.executeVerification(context.Background(), child, logs)
	if err != nil || passed || len(checks) != 2 || checks[0].ExitCode != 0 || checks[0].VerificationProblem == "" || checks[1].Command != "npm run typecheck" {
		t.Fatalf("new batch downgraded zero-test protection: auto=%v checks=%+v passed=%v error=%v", child.AutoChecks, checks, passed, err)
	}
	output, err := os.ReadFile(filepath.Join(logs, "check-2.log"))
	if err != nil || !strings.Contains(string(output), "fresh-typecheck-ran") {
		t.Fatalf("the newly discovered check was not actually executed: %s %v", output, err)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.4
func TestExhaustedBatchAncestrySurvivesArchiveAndRejectsLostReceipts(t *testing.T) {
	for _, tamper := range []bool{false, true} {
		t.Run(fmt.Sprintf("tamper-%t", tamper), func(t *testing.T) {
			agent := &fixtureAgent{}
			e, parent := preparedExhaustedTask(t, agent)
			mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained source\n")
			child := claimExhaustedChild(t, e, parent, "first-generation")
			if err := e.prepareExhaustedSeed(context.Background(), child); err != nil {
				t.Fatal(err)
			}
			if err := e.ArchiveTask(parent.ID, true); err != nil {
				t.Fatal(err)
			}
			child, err := e.Store.Update(child.ID, func(saved *Task) error {
				saved.Round, saved.State, saved.EvidenceHash = MaxRounds, stateExhausted, ""
				saved.ContinuePending = false
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if tamper {
				if err = os.Remove(filepath.Join(e.Root, "evidence", child.ID, child.SeedEvidence[1])); err != nil {
					t.Fatal(err)
				}
			}
			grandchild := claimExhaustedChild(t, e, child, "second-generation")
			err = e.RunTask(context.Background(), grandchild.ID)
			got, _ := e.Store.Get(grandchild.ID)
			if tamper {
				if err == nil || !strings.Contains(err.Error(), "batch_seed_invalid") || got.Round != 0 || agent.total != 0 {
					t.Fatalf("missing ancestor evidence was ignored: state=%s round=%d calls=%d error=%v", got.State, got.Round, agent.total, err)
				}
			} else if err != nil || got.State != stateAwaitingAcceptance || got.Round != 1 {
				t.Fatalf("archived ancestor blocked a valid next batch: state=%s round=%d error=%v", got.State, got.Round, err)
			}
		})
	}
}

// @covers AC-RELIABLE-RENEWAL-001.6
func TestExhaustedBatchAcceptanceRejectsChangedAncestorReceipt(t *testing.T) {
	e, parent := preparedExhaustedTask(t, &fixtureAgent{})
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained source\n")
	child := claimExhaustedChild(t, e, parent, "first")
	if err := e.prepareExhaustedSeed(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	child, err := e.Store.Update(child.ID, func(saved *Task) error {
		saved.State, saved.Round = stateExhausted, MaxRounds
		saved.ContinuePending = false
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	grandchild := claimExhaustedChild(t, e, child, "second")
	if err = e.RunTask(context.Background(), grandchild.ID); err != nil {
		t.Fatal(err)
	}
	grandchild, _ = e.Store.Get(grandchild.ID)
	if _, err = e.acceptanceEvidence(context.Background(), grandchild); err != nil {
		t.Fatalf("positive control failed before ancestor mutation: %v", err)
	}
	name := filepath.Join(e.Root, "evidence", child.ID, child.SeedEvidence[1])
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(name, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.acceptanceEvidence(context.Background(), grandchild); err == nil {
		t.Fatal("changed ancestor inheritance receipt was ignored at acceptance")
	}
}
