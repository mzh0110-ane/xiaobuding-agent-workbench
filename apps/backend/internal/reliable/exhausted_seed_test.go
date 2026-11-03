package reliable

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// @covers AC-RELIABLE-RENEWAL-001.1
func TestExhaustedBatchExplicitContinuation(t *testing.T) {
	agent := &recoveryAuditAgent{expected: "retained work\n"}
	e, parent := preparedCancelledTask(t, agent, MaxRounds)
	parent, err := e.Store.Update(parent.ID, func(saved *Task) error {
		saved.State = stateExhausted
		saved.Reason = "The independent reviewer found missing behavior"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(parent.Worktree, "app.txt"), []byte("retained work\n"), 0600); err != nil {
		t.Fatal(err)
	}
	h := &HTTP{Engine: e, Token: "test-action"}
	r := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+parent.ID+"/exhausted-next-batch", `{"id":"continue-one","kind":"requirement","content":"Finish the review findings"}`, h.Token)
	if r.Code != http.StatusAccepted {
		t.Fatalf("continuation rejected: %d %s", r.Code, r.Body)
	}
	var message TaskMessage
	if err = json.Unmarshal(r.Body.Bytes(), &message); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	child, err := e.Store.Get(message.LinkedTask)
	if err != nil {
		t.Fatal(err)
	}
	if child.State != stateAwaitingAcceptance || child.Round != 1 || child.ParentTaskID != parent.ID || child.Worktree == parent.Worktree || child.Base != parent.Base {
		t.Fatalf("wrong successor: state=%s round=%d parent=%s worktree=%s reason=%s", child.State, child.Round, child.ParentTaskID, child.Worktree, child.Reason)
	}
	old, _ := e.Store.Get(parent.ID)
	if old.State != stateExhausted || old.Round != MaxRounds {
		t.Fatal("continuation reset the exhausted parent")
	}
	for _, check := range []struct{ dir, want string }{{parent.Project, "before\n"}, {parent.Worktree, "retained work\n"}} {
		raw, readErr := os.ReadFile(filepath.Join(check.dir, "app.txt"))
		if readErr != nil || string(raw) != check.want {
			t.Fatalf("continuation modified a retained source: %s %s %v", check.dir, raw, readErr)
		}
	}
	if _, err = e.acceptanceEvidence(context.Background(), child); err != nil {
		t.Fatalf("new batch evidence did not seal: %v", err)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.1
func TestExhaustedBatchRejectsQuestionKindWithoutCreatingSuccessor(t *testing.T) {
	agent := &fixtureAgent{}
	e, parent := preparedExhaustedTask(t, agent)
	h := &HTTP{Engine: e, Token: "test-action"}
	r := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+parent.ID+"/exhausted-next-batch", `{"id":"question","kind":"question","content":"What is the remaining work?"}`, h.Token)
	e.wg.Wait()
	tasks, err := e.Store.List()
	if err != nil {
		t.Fatal(err)
	}
	if r.Code != http.StatusConflict || len(tasks) != 1 || agent.total != 0 {
		t.Fatalf("read-only question created writable work: status=%d tasks=%d calls=%d body=%s", r.Code, len(tasks), agent.total, r.Body)
	}
}
