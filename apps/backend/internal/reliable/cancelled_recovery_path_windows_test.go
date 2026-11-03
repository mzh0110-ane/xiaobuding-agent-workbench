package reliable

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCancelledRecoveryRejectsRedirectedWorktree(t *testing.T) {
	agent := &fixtureAgent{}
	engine, task := preparedCancelledTask(t, agent, 1)
	external := filepath.Join(t.TempDir(), "retained")
	if err := os.Rename(task.Worktree, external); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("cmd.exe", "/c", "mklink", "/J", task.Worktree, external).CombinedOutput()
	if err != nil {
		t.Fatalf("create test junction: %v: %s", err, output)
	}
	t.Cleanup(func() {
		if err := os.Remove(task.Worktree); err != nil {
			t.Error(err)
		}
	})
	if code, body := recoveryHTTP(engine, task.ID); code != http.StatusAccepted {
		t.Fatalf("claim rejected before candidate validation: %d %s", code, body)
	}
	engine.wg.Wait()
	saved, err := engine.Store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.State != stateCancelled || agent.total != 0 || !strings.Contains(saved.Reason, "recovery_candidate_invalid") {
		t.Fatalf("redirected working copy admitted a writer: state=%s calls=%d reason=%s", saved.State, agent.total, saved.Reason)
	}
	raw, err := os.ReadFile(filepath.Join(external, "app.txt"))
	if err != nil || string(raw) != "before\n" {
		t.Fatalf("redirected source was modified: %q: %v", raw, err)
	}
}
