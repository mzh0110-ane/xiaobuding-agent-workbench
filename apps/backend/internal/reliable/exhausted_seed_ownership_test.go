package reliable

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func interruptedExhaustedSeed(t *testing.T, agent AgentRunner) (*Engine, *Task, *Task) {
	t.Helper()
	e, parent := preparedExhaustedTask(t, agent)
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained work\n")
	child := claimExhaustedChild(t, e, parent, "owned-retry")
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
	child, err = e.Store.Update(child.ID, func(saved *Task) error { saved.State = stateInterrupted; return nil })
	if err != nil {
		t.Fatal(err)
	}
	return e, parent, child
}

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedPendingSeedRejectsUnownedBranchBeforeCopy(t *testing.T) {
	agent := &fixtureAgent{}
	e, _, child := interruptedExhaustedSeed(t, agent)
	gitTest(t, child.Worktree, "checkout", "--detach")
	err := e.RunTask(context.Background(), child.ID)
	raw, readErr := os.ReadFile(filepath.Join(child.Worktree, "app.txt"))
	if err == nil || !strings.Contains(err.Error(), "recovery_candidate_invalid") || agent.total != 0 || readErr != nil || string(raw) != "before\n" {
		t.Fatalf("unowned pending candidate was copied into: calls=%d bytes=%s read=%v error=%v", agent.total, raw, readErr, err)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.4
func TestExhaustedBatchCancelledImmediatelyAfterAdmissionCanRecover(t *testing.T) {
	e, parent := preparedExhaustedTask(t, &fixtureAgent{})
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained source\n")
	child := claimExhaustedChild(t, e, parent, "immediate-cancel")
	if err := e.Cancel(child.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RecoverCancelled(child.ID); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	got, _ := e.Store.Get(child.ID)
	if got.State != stateAwaitingAcceptance || got.Round != 1 || got.SeedPending != nil || got.RecoveryPending != nil {
		t.Fatalf("stopping before initial preparation stranded the new batch: state=%s round=%d reason=%s", got.State, got.Round, got.Reason)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedPendingSeedRejectsUnknownChildProcessCleanup(t *testing.T) {
	agent := &fixtureAgent{}
	e, parent := preparedExhaustedTask(t, agent)
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained source\n")
	child := claimExhaustedChild(t, e, parent, "unknown-child")
	if _, err := e.Store.Update(child.ID, func(saved *Task) error {
		saved.ProcessCleanupEpoch = randomID()
		saved.Reason = "owned_process_cleanup: the previous candidate process was not confirmed stopped"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err := e.RunTask(context.Background(), child.ID)
	got, _ := e.Store.Get(child.ID)
	if err == nil || !strings.Contains(err.Error(), "recovery_process_unknown") || agent.total != 0 || got.Worktree != "" {
		t.Fatalf("pending seed bypassed unknown candidate ownership: calls=%d worktree=%s error=%v", agent.total, got.Worktree, err)
	}
}
