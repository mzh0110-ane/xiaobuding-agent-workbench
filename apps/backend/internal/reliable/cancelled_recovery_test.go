package reliable

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type recoveryAuditAgent struct {
	fixtureAgent
	expected string
	seen     chan string
	gate     chan struct{}
}

func (a *recoveryAuditAgent) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	if request.Role == roleAuditor {
		raw, err := os.ReadFile(filepath.Join(request.Directory, "app.txt"))
		if err != nil {
			return nil, err
		}
		if a.expected != "" && string(raw) != a.expected {
			return nil, fmt.Errorf("retained partial source lost before audit: %s", raw)
		}
		if a.seen != nil {
			a.seen <- string(raw)
			select {
			case <-a.gate:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	return a.fixtureAgent.Run(ctx, request)
}

// @covers AC-RELIABLE-RECOVERY-001.2
// @covers AC-RELIABLE-RECOVERY-001.5
func TestCancelledRecoveryPreservesPartialSource(t *testing.T) {
	agent := &recoveryAuditAgent{}
	e, task := fixtureEngine(t, agent, passingCheck)
	t.Cleanup(func() { _ = e.Close() })
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.Cancel(task.ID); err != nil {
		t.Fatal(err)
	}
	stopped, _ := e.Store.Get(task.ID)
	oldPatch := filepath.Join(e.Root, "evidence", task.ID, "round-1", "changes.patch")
	oldBytes, err := os.ReadFile(oldPatch)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(stopped.Worktree, "app.txt"), []byte("partial work retained\n"), 0600); err != nil {
		t.Fatal(err)
	}
	agent.expected = "partial work retained\n"
	h := &HTTP{Engine: e, Token: "test-action"}
	response := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+task.ID+"/recover-cancelled", "{}", h.Token)
	if response.Code != http.StatusAccepted {
		t.Fatalf("recovery rejected: %d %s", response.Code, response.Body)
	}
	e.wg.Wait()
	got, _ := e.Store.Get(task.ID)
	if got.State != stateAwaitingAcceptance || got.Round != 2 || got.Worktree != stopped.Worktree || agent.implement != 2 {
		t.Fatalf("wrong recovered result: state=%s round=%d worktree=%s calls=%d reason=%s", got.State, got.Round, got.Worktree, agent.implement, got.Reason)
	}
	if original, _ := os.ReadFile(filepath.Join(task.Project, "app.txt")); string(original) != "before\n" {
		t.Fatal("recovery applied the candidate without acceptance")
	}
	if current, _ := os.ReadFile(oldPatch); string(current) != string(oldBytes) {
		t.Fatal("recovery rewrote prior-round patch")
	}
	if _, err = e.acceptanceEvidence(context.Background(), got); err != nil {
		t.Fatalf("recovered evidence did not seal: %v", err)
	}
	if got.RecoveryPending != nil || len(got.RecoveryEvidence) < 2 {
		t.Fatal("recovery provenance was not durably published")
	}
	receipt := filepath.Join(e.Root, "evidence", task.ID, got.RecoveryEvidence[1])
	raw, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(receipt, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.acceptanceEvidence(context.Background(), got); err == nil {
		t.Fatal("altered recovery receipt did not invalidate acceptance")
	}
}

func preparedCancelledTask(t *testing.T, agent AgentRunner, round int) (*Engine, *Task) {
	t.Helper()
	e, task := fixtureEngine(t, agent, passingCheck)
	t.Cleanup(func() { _ = e.Close() })
	if err := e.prepare(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	saved, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.State, saved.Round = stateCancelled, round
		saved.ApprovedPaths = []string{"app.txt"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return e, saved
}

func recoveryHTTP(e *Engine, id string) (int, string) {
	h := &HTTP{Engine: e, Token: "test-action"}
	r := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+id+"/recover-cancelled", "{}", h.Token)
	return r.Code, r.Body.String()
}

// @covers AC-RELIABLE-RECOVERY-001.3
func TestCancelledRecoveryRejectsDrift(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		mutate func(*testing.T, *Engine, *Task)
	}{
		{"original", "recovery_original_changed", func(t *testing.T, _ *Engine, task *Task) {
			mustRecoveryWrite(t, filepath.Join(task.Project, "app.txt"), "outside edit")
		}},
		{"hidden-original", "recovery_original_changed", func(t *testing.T, _ *Engine, task *Task) {
			gitTest(t, task.Project, "update-index", "--assume-unchanged", "app.txt")
			mustRecoveryWrite(t, filepath.Join(task.Project, "app.txt"), "outside edit")
		}},
		{"contract", "recovery_evidence_invalid", func(t *testing.T, e *Engine, task *Task) {
			if err := os.Remove(filepath.Join(e.Root, "evidence", task.ID, "contract.json")); err != nil {
				t.Fatal(err)
			}
		}},
		{"baseline", "recovery_evidence_invalid", func(t *testing.T, e *Engine, task *Task) {
			_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.Baseline.Hash = "tampered"; return nil })
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"candidate-branch", "recovery_candidate_invalid", func(t *testing.T, _ *Engine, task *Task) {
			gitTest(t, task.Worktree, "checkout", "--detach")
		}},
		{"scope", "recovery_scope", func(t *testing.T, _ *Engine, task *Task) {
			mustRecoveryWrite(t, filepath.Join(task.Worktree, "unapproved.txt"), "retained, not discarded")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			agent := &fixtureAgent{}
			e, task := preparedCancelledTask(t, agent, 1)
			mustRecoveryWrite(t, filepath.Join(task.Worktree, "app.txt"), "partial\n")
			test.mutate(t, e, task)
			if code, body := recoveryHTTP(e, task.ID); code != http.StatusAccepted {
				t.Fatalf("claim failed before validation: %d %s", code, body)
			}
			e.wg.Wait()
			saved, _ := e.Store.Get(task.ID)
			if saved.State != stateCancelled || saved.Round != 1 || !strings.Contains(saved.Reason, test.prefix) || agent.total != 0 {
				t.Fatalf("invalid recovery advanced: state=%s round=%d calls=%d reason=%s", saved.State, saved.Round, agent.total, saved.Reason)
			}
			if data, _ := os.ReadFile(filepath.Join(task.Worktree, "app.txt")); string(data) != "partial\n" {
				t.Fatal("validation destroyed partial work")
			}
		})
	}
}

func mustRecoveryWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

// @covers AC-RELIABLE-RECOVERY-001.4
func TestCancelledRecoverySingleOwnerAndCancellation(t *testing.T) {
	agent := &recoveryAuditAgent{seen: make(chan string, 1), gate: make(chan struct{})}
	e, task := preparedCancelledTask(t, agent, 1)
	if code, body := recoveryHTTP(e, task.ID); code != http.StatusAccepted {
		t.Fatalf("recovery not admitted: %d %s", code, body)
	}
	select {
	case <-agent.seen:
	case <-time.After(30 * time.Second):
		t.Fatal("recovery did not reach its fresh audit")
	}
	if code, _ := recoveryHTTP(e, task.ID); code != http.StatusConflict {
		t.Fatal("second recovery click admitted another execution")
	}
	if err := e.ArchiveTask(task.ID, true); err == nil {
		t.Fatal("active recovery was archived")
	}
	if err := e.Store.Create(&Task{ID: "conflict", Project: task.Project, State: stateQueued}); err == nil {
		t.Fatal("active recovered project lost its ownership")
	}
	if err := e.Cancel(task.ID); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	saved, _ := e.Store.Get(task.ID)
	if saved.State != stateCancelled || saved.Round != 1 || agent.implement != 0 {
		t.Fatal("cancelling the new audit spent an implementation round")
	}
}

// @covers AC-RELIABLE-RECOVERY-001.6
func TestCancelledRecoveryPendingGuard(t *testing.T) {
	agent := &fixtureAgent{}
	e, task := preparedCancelledTask(t, agent, 1)
	if err := e.saveJSON(task.ID, "prior-recovery.json", recoveryRecord(task, "prior", task.Baseline)); err != nil {
		t.Fatal(err)
	}
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.RecoveryEvidence = []string{"prior-recovery.json"}
		saved.State = stateAuditing
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Store.Recover(); err != nil {
		t.Fatal(err)
	}
	mustRecoveryWrite(t, filepath.Join(task.Project, "app.txt"), "drift after receipt publication")
	if err = e.RunTask(context.Background(), task.ID); err == nil {
		t.Fatal("restart bypassed recovery validation")
	}
	saved, _ := e.Store.Get(task.ID)
	if saved.State != stateCancelled || saved.Round != 1 || agent.total != 0 || !strings.Contains(saved.Reason, "recovery_original_changed") {
		t.Fatalf("wrong restart validation: %+v", saved)
	}
}

// @covers AC-RELIABLE-RECOVERY-001.5
func TestCancelledRecoveryKeepsRoundBudget(t *testing.T) {
	for _, round := range []int{0, 2} {
		t.Run(fmt.Sprint(round), func(t *testing.T) {
			agent := &fixtureAgent{}
			e, task := preparedCancelledTask(t, agent, round)
			if code, body := recoveryHTTP(e, task.ID); code != http.StatusAccepted {
				t.Fatalf("recovery not admitted: %d %s", code, body)
			}
			e.wg.Wait()
			saved, _ := e.Store.Get(task.ID)
			if saved.Round != round+1 || saved.State != stateAwaitingAcceptance || agent.implement != 1 {
				t.Fatalf("round budget changed: round=%d state=%s reason=%s", saved.Round, saved.State, saved.Reason)
			}
		})
	}
}

// @covers AC-RELIABLE-RECOVERY-001.1
// @covers AC-RELIABLE-RECOVERY-001.4
func TestCancelledRecoveryAdmission(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*Task)
	}{
		{"readonly", func(task *Task) { task.ReadOnly = true }},
		{"archived", func(task *Task) { task.ArchivedAt = "2026-10-20T00:00:00Z" }},
		{"exhausted", func(task *Task) { task.Round = 3 }},
		{"stopping", func(task *Task) { task.State = stateCancelling }},
	} {
		t.Run(test.name, func(t *testing.T) {
			agent := &fixtureAgent{}
			e, task := fixtureEngine(t, agent, passingCheck)
			t.Cleanup(func() { _ = e.Close() })
			_, _ = e.Store.Update(task.ID, func(task *Task) error { task.State = stateCancelled; test.edit(task); return nil })
			h := &HTTP{Engine: e, Token: "test-action"}
			if r := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+task.ID+"/recover-cancelled", "{}", ""); r.Code != http.StatusForbidden {
				t.Fatalf("unauthorized recovery: %d", r.Code)
			}
			r := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+task.ID+"/recover-cancelled", "{}", h.Token)
			if r.Code != http.StatusConflict || !strings.Contains(r.Body.String(), "recovery_") || agent.total != 0 {
				t.Fatalf("bad admission: %d %s", r.Code, r.Body)
			}
		})
	}
}

// @covers AC-RELIABLE-RECOVERY-001.4
func TestCancelledRecoveryConflicts(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(func() { _ = e.Close() })
	if err := e.Cancel(task.ID); err != nil {
		t.Fatal(err)
	}
	other := &Task{ID: "new-owner", Project: task.Project, Goal: "Other work", State: stateQueued}
	if err := e.Store.Create(other); err != nil {
		t.Fatal(err)
	}
	h := &HTTP{Engine: e, Token: "test-action"}
	r := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+task.ID+"/recover-cancelled", "{}", h.Token)
	if r.Code != http.StatusConflict || !strings.Contains(r.Body.String(), "recovery_project_conflict") {
		t.Fatalf("conflict missed: %d %s", r.Code, r.Body)
	}
	saved, _ := e.Store.Get(task.ID)
	if saved.State != stateCancelled || saved.Round != 0 {
		t.Fatal("failed recovery changed task admission")
	}
	if _, err := json.Marshal(saved); err != nil {
		t.Fatal(err)
	}
}
