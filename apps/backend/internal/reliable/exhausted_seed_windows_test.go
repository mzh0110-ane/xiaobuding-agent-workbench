package reliable

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedPendingSeedRejectsRedirectedWorktreeBeforeCopy(t *testing.T) {
	agent := &fixtureAgent{}
	e, _, child := interruptedExhaustedSeed(t, agent)
	external := filepath.Join(t.TempDir(), "retained")
	if err := os.Rename(child.Worktree, external); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("cmd.exe", "/c", "mklink", "/J", child.Worktree, external).CombinedOutput()
	if err != nil {
		t.Fatalf("create fixture junction: %v: %s", err, out)
	}
	t.Cleanup(func() {
		if err := os.Remove(child.Worktree); err != nil {
			t.Error(err)
		}
	})
	err = e.RunTask(context.Background(), child.ID)
	raw, readErr := os.ReadFile(filepath.Join(external, "app.txt"))
	if err == nil || !strings.Contains(err.Error(), "recovery_candidate_invalid") || agent.total != 0 || readErr != nil || string(raw) != "before\n" {
		t.Fatalf("redirected pending source was modified: calls=%d bytes=%s read=%v error=%v", agent.total, raw, readErr, err)
	}
}
