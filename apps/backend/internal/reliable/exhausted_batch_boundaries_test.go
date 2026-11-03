package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func preparedExhaustedTask(t *testing.T, agent AgentRunner) (*Engine, *Task) {
	t.Helper()
	e, parent := preparedCancelledTask(t, agent, MaxRounds)
	parent, err := e.Store.Update(parent.ID, func(saved *Task) error {
		saved.State = stateExhausted
		saved.Reason = "Review findings remain unresolved"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return e, parent
}

func claimExhaustedChild(t *testing.T, e *Engine, parent *Task, id string) *Task {
	t.Helper()
	message, err := e.StartExhaustedBatch(parent.ID, messageRequest{ID: id, Kind: messageRequirement, Content: "Continue the project"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := e.Store.Get(message.LinkedTask)
	if err != nil {
		t.Fatal(err)
	}
	return child
}

// @covers AC-RELIABLE-RENEWAL-001.2
func TestExhaustedBatchAdmissionIsAtomicAndIdempotent(t *testing.T) {
	e, parent := preparedExhaustedTask(t, &fixtureAgent{})
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, id := range []string{"first", "second"} {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := e.StartExhaustedBatch(parent.ID, messageRequest{ID: id})
			results <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !strings.Contains(err.Error(), "batch_successor_exists") {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatalf("created %d successor claims", success)
	}
	old, _ := e.Store.Get(parent.ID)
	if len(old.Messages) != 1 || old.SuccessorTaskID == "" {
		t.Fatal("atomic claim lost its durable link")
	}
	message := old.Messages[0]
	again, err := e.StartExhaustedBatch(parent.ID, messageRequest{ID: message.ID})
	if err != nil || again.LinkedTask != old.SuccessorTaskID {
		t.Fatalf("retry was not idempotent: %+v %v", again, err)
	}
	other := &Task{ID: "unrelated", Project: parent.Project, State: stateQueued}
	if err = e.Store.Create(other); err == nil {
		t.Fatal("a pending successor did not retain exclusive project ownership")
	}
}

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedBatchRejectsInvalidParentBeforeAgentCall(t *testing.T) {
	tests := []struct {
		name string
		edit func(*testing.T, *Engine, *Task)
		code string
	}{
		{"original-drift", func(t *testing.T, _ *Engine, parent *Task) {
			mustRecoveryWrite(t, filepath.Join(parent.Project, "app.txt"), "outside change")
		}, "recovery_original_changed"},
		{"hidden-original-drift", func(t *testing.T, _ *Engine, parent *Task) {
			gitTest(t, parent.Project, "update-index", "--assume-unchanged", "app.txt")
			mustRecoveryWrite(t, filepath.Join(parent.Project, "app.txt"), "hidden outside change")
		}, "recovery_original_changed"},
		{"scope", func(t *testing.T, _ *Engine, parent *Task) {
			mustRecoveryWrite(t, filepath.Join(parent.Worktree, "unknown.txt"), "outside scope")
		}, "recovery_scope"},
		{"wrong-branch", func(t *testing.T, _ *Engine, parent *Task) {
			gitTest(t, parent.Worktree, "checkout", "--detach")
		}, "recovery_candidate_invalid"},
		{"missing-contract", func(t *testing.T, e *Engine, parent *Task) {
			if err := os.Remove(filepath.Join(e.Root, "evidence", parent.ID, "contract.json")); err != nil {
				t.Fatal(err)
			}
		}, "recovery_evidence_invalid"},
		{"unknown-process", func(t *testing.T, e *Engine, parent *Task) {
			_, err := e.Store.Update(parent.ID, func(saved *Task) error {
				saved.Reason = "owned_process_cleanup: process exit was not confirmed"
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}, "recovery_process_unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			agent := &fixtureAgent{}
			e, parent := preparedExhaustedTask(t, agent)
			mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained work\n")
			test.edit(t, e, parent)
			parent, _ = e.Store.Get(parent.ID)
			child := claimExhaustedChild(t, e, parent, "request")
			err := e.RunTask(context.Background(), child.ID)
			got, _ := e.Store.Get(child.ID)
			if err == nil || !strings.Contains(err.Error(), test.code) || got.Round != 0 || agent.total != 0 {
				t.Fatalf("unsafe parent advanced: state=%s round=%d calls=%d error=%v", got.State, got.Round, agent.total, err)
			}
			if got.SeedPending == nil {
				t.Fatal("validation failure discarded the durable continuation obligation")
			}
		})
	}
}

// @covers AC-RELIABLE-RENEWAL-001.4
func TestExhaustedBatchRestartResumesPartiallyCopiedSource(t *testing.T) {
	agent := &recoveryAuditAgent{expected: "retained work\n"}
	e, parent := preparedExhaustedTask(t, agent)
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained work\n")
	child := claimExhaustedChild(t, e, parent, "restart")
	snap, err := TakeSnapshot(context.Background(), parent.Worktree)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.ensureExhaustedIntent(child, child.SeedPending, snap); err != nil {
		t.Fatal(err)
	}
	if err = e.ensureSeedModelPlan(child, child.SeedPending); err != nil {
		t.Fatal(err)
	}
	if err = e.prepare(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	mustRecoveryWrite(t, filepath.Join(child.Worktree, "app.txt"), "retained work\n")
	if err = e.Store.Recover(); err != nil {
		t.Fatal(err)
	}
	if err = e.RunTask(context.Background(), child.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.Store.Get(child.ID)
	if got.Round != 1 || got.State != stateAwaitingAcceptance || got.SeedPending != nil || len(got.SeedEvidence) != 2 {
		t.Fatalf("partial seed did not resume: state=%s round=%d pending=%+v", got.State, got.Round, got.SeedPending)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.4
func TestExhaustedBatchRestartRejectsUnexpectedCandidateEdits(t *testing.T) {
	agent := &fixtureAgent{}
	e, parent := preparedExhaustedTask(t, agent)
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained work\n")
	child := claimExhaustedChild(t, e, parent, "preserve")
	snap, _ := TakeSnapshot(context.Background(), parent.Worktree)
	if _, err := e.ensureExhaustedIntent(child, child.SeedPending, snap); err != nil {
		t.Fatal(err)
	}
	if err := e.ensureSeedModelPlan(child, child.SeedPending); err != nil {
		t.Fatal(err)
	}
	if err := e.prepare(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	mustRecoveryWrite(t, filepath.Join(child.Worktree, "app.txt"), "manual edit after interruption\n")
	if _, err := e.Store.Update(child.ID, func(saved *Task) error { saved.State = stateInterrupted; return nil }); err != nil {
		t.Fatal(err)
	}
	err := e.RunTask(context.Background(), child.ID)
	raw, _ := os.ReadFile(filepath.Join(child.Worktree, "app.txt"))
	if err == nil || !strings.Contains(err.Error(), "batch_seed_invalid") || agent.total != 0 || string(raw) != "manual edit after interruption\n" {
		t.Fatalf("unexpected edits were overwritten: error=%v calls=%d bytes=%s", err, agent.total, raw)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.5
func TestExhaustedSeedReceiptRemainsAnExecutionAndAcceptanceGate(t *testing.T) {
	agent := &fixtureAgent{}
	e, parent := preparedExhaustedTask(t, agent)
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained work\n")
	child := claimExhaustedChild(t, e, parent, "tamper")
	if err := e.RunTask(context.Background(), child.ID); err != nil {
		t.Fatal(err)
	}
	child, _ = e.Store.Get(child.ID)
	name := filepath.Join(e.Root, "evidence", child.ID, child.SeedEvidence[1])
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(name, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.acceptanceEvidence(context.Background(), child); err == nil {
		t.Fatal("changed inheritance evidence did not invalidate acceptance")
	}
	var receipt map[string]any
	if err = json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	receipt["source_hash"] = "changed"
	raw, _ = json.Marshal(receipt)
	if err = os.WriteFile(name, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Store.Update(child.ID, func(saved *Task) error {
		saved.State, saved.EvidenceHash = stateInterrupted, ""
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before := agent.total
	err = e.RunTask(context.Background(), child.ID)
	if err == nil || !strings.Contains(err.Error(), "batch_seed_invalid") || agent.total != before {
		t.Fatalf("execution skipped its durable inheritance guard: %v", err)
	}
}
