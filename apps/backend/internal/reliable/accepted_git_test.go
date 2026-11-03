package reliable

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func acceptedGitFixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	project := fixtureRepo(t)
	data := t.TempDir()
	work, base, err := PrepareWorktree(context.Background(), data, project, "candidate")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(work, "app.txt"), []byte("reviewed\n"), 0600))
	patch := filepath.Join(data, "changes.patch")
	require.NoError(t, ExportPatch(context.Background(), work, base, patch))
	return project, base, patch, data
}

func TestAcceptedGitAppliesAndCheckpointsOnlyReviewedPatch(t *testing.T) {
	project, base, patch, data := acceptedGitFixture(t)
	var saved *AcceptanceCheckpoint
	persist := func(value *AcceptanceCheckpoint) error { copy := *value; saved = &copy; return nil }
	got, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, &AcceptanceCheckpoint{Base: base}, persist, true)
	require.NoError(t, err)
	require.True(t, got.Complete)
	require.Equal(t, got.Commit+"\n", gitTest(t, project, "rev-parse", "HEAD"))
	require.Empty(t, gitTest(t, project, "status", "--porcelain=v1", "--untracked-files=all"))
	require.Equal(t, "2\n", gitTest(t, project, "rev-list", "--count", "HEAD"))
	again, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, saved, persist, false)
	require.NoError(t, err)
	require.Equal(t, got.Commit, again.Commit)
	require.Equal(t, "2\n", gitTest(t, project, "rev-list", "--count", "HEAD"))
	files, err := os.ReadDir(project)
	require.NoError(t, err)
	require.Len(t, files, 2) // .git and app.txt only
}

func TestAcceptedGitPreservesManualChangesAndStaging(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(map[bool]string{false: "unstaged", true: "staged"}[staged], func(t *testing.T) {
			project, base, patch, data := acceptedGitFixture(t)
			require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("manual\n"), 0600))
			if staged {
				gitTest(t, project, "add", "app.txt")
			}
			before, err := os.ReadFile(filepath.Join(project, ".git", "index"))
			require.NoError(t, err)
			_, err = checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, func(*AcceptanceCheckpoint) error { return nil }, true)
			require.Error(t, err)
			after, err := os.ReadFile(filepath.Join(project, ".git", "index"))
			require.NoError(t, err)
			require.Equal(t, before, after)
			require.Equal(t, base+"\n", gitTest(t, project, "rev-parse", "HEAD"))
			raw, err := os.ReadFile(filepath.Join(project, "app.txt"))
			require.NoError(t, err)
			require.Equal(t, "manual\n", string(raw))
		})
	}
}

func TestAcceptedGitRecoversAfterCommitAndBeforeIndexInstall(t *testing.T) {
	project, base, patch, data := acceptedGitFixture(t)
	var saved *AcceptanceCheckpoint
	persist := func(value *AcceptanceCheckpoint) error {
		copy := *value
		saved = &copy
		if value.Commit != "" && !value.Complete {
			// Model a process stopping after the durable commit intent, with ref
			// update completed and the original index still unchanged.
			gitTest(t, project, "update-ref", value.Reference, value.Commit, value.Base)
			return errors.New("injected stop before index install")
		}
		return nil
	}
	_, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, persist, true)
	require.ErrorContains(t, err, "injected stop")
	require.NotNil(t, saved)
	got, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, saved, func(value *AcceptanceCheckpoint) error { copy := *value; saved = &copy; return nil }, false)
	require.NoError(t, err)
	require.True(t, got.Complete)
	require.Empty(t, gitTest(t, project, "status", "--porcelain=v1", "--untracked-files=all"))
	require.Equal(t, "2\n", gitTest(t, project, "rev-list", "--count", "HEAD"))
}

func TestAcceptedGitReusesUserCommitAndRejectsForeignLock(t *testing.T) {
	project, base, patch, data := acceptedGitFixture(t)
	require.NoError(t, ApplyPatch(context.Background(), project, base, patch))
	gitTest(t, project, "add", "app.txt")
	gitTest(t, project, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "user saved accepted work")
	head := strings.TrimSpace(gitTest(t, project, "rev-parse", "HEAD"))
	got, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, func(*AcceptanceCheckpoint) error { return nil }, false)
	require.NoError(t, err)
	require.Equal(t, head, got.Commit)
	require.Equal(t, "2\n", gitTest(t, project, "rev-list", "--count", "HEAD"))
	require.NoError(t, os.WriteFile(filepath.Join(project, ".git", "index.lock"), []byte("foreign lock"), 0600))
	_, err = checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, func(*AcceptanceCheckpoint) error { return nil }, false)
	require.ErrorContains(t, err, "local_checkpoint_locked")
	raw, err := os.ReadFile(filepath.Join(project, ".git", "index.lock"))
	require.NoError(t, err)
	require.Equal(t, "foreign lock", string(raw))
}

func TestAcceptedGitLegacyRequiresExactAppliedBytes(t *testing.T) {
	project, base, patch, data := acceptedGitFixture(t)
	_, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, func(*AcceptanceCheckpoint) error { return nil }, false)
	require.ErrorContains(t, err, "local_checkpoint_drift")
	raw, err := os.ReadFile(filepath.Join(project, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, "before\n", string(raw))
	require.NoError(t, ApplyPatch(context.Background(), project, base, patch))
	gitTest(t, project, "add", "app.txt") // Even exact accepted edits staged by the user are preserved.
	before, err := os.ReadFile(filepath.Join(project, ".git", "index"))
	require.NoError(t, err)
	_, err = checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, func(*AcceptanceCheckpoint) error { return nil }, false)
	require.ErrorContains(t, err, "local_checkpoint_staged")
	after, err := os.ReadFile(filepath.Join(project, ".git", "index"))
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, base+"\n", gitTest(t, project, "rev-parse", "HEAD"))
}

func TestAcceptedGitNoOpReusesBaseAndGitOperationBlocks(t *testing.T) {
	project := fixtureRepo(t)
	base := strings.TrimSpace(gitTest(t, project, "rev-parse", "HEAD"))
	data := t.TempDir()
	patch := filepath.Join(data, "empty.patch")
	require.NoError(t, os.WriteFile(patch, nil, 0600))
	got, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, func(*AcceptanceCheckpoint) error { return nil }, true)
	require.NoError(t, err)
	require.True(t, got.Complete)
	require.Equal(t, base, got.Commit)
	require.Equal(t, "1\n", gitTest(t, project, "rev-list", "--count", "HEAD"))
	for _, name := range []string{"MERGE_HEAD", "BISECT_START", "rebase-merge"} {
		path := filepath.Join(project, ".git", name)
		require.NoError(t, os.WriteFile(path, []byte("operation state"), 0600))
		_, err = checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, func(*AcceptanceCheckpoint) error { return nil }, true)
		require.ErrorContains(t, err, "local_checkpoint_git_operation")
		raw, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, "operation state", string(raw))
		require.NoError(t, os.Remove(path))
	}
}

func TestAcceptedGitNewBinaryDeletionAndOwnedLockRecovery(t *testing.T) {
	project, base, patch, data := acceptedGitFixture(t)
	work := filepath.Join(data, "worktrees", "candidate")
	require.NoError(t, os.Remove(filepath.Join(work, "app.txt")))
	name := "中文 空格.bin"
	content := []byte{0, 255, 0, 1, 2, 13, 10, 255}
	require.NoError(t, os.WriteFile(filepath.Join(work, name), content, 0600))
	require.NoError(t, ExportPatch(context.Background(), work, base, patch))
	var saved *AcceptanceCheckpoint
	_, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, nil, func(value *AcceptanceCheckpoint) error {
		copy := *value
		saved = &copy
		if value.Commit != "" {
			return errors.New("stop with known intent")
		}
		return nil
	}, true)
	require.ErrorContains(t, err, "stop with known intent")
	// Recreate the exact marker a terminated process would leave. A fresh
	// controller can recognize its own journal without deleting foreign locks.
	lock := filepath.Join(project, ".git", "index.lock")
	require.NoError(t, os.WriteFile(lock, []byte("xiaobuding-accepted-checkpoint:"+saved.Nonce+"\n"), 0600))
	got, err := checkpointAcceptedPatch(context.Background(), project, base, patch, data, saved, func(*AcceptanceCheckpoint) error { return nil }, false)
	require.NoError(t, err)
	require.True(t, got.Complete)
	_, err = os.Stat(filepath.Join(project, "app.txt"))
	require.True(t, os.IsNotExist(err))
	raw, err := os.ReadFile(filepath.Join(project, name))
	require.NoError(t, err)
	require.Equal(t, content, raw)
	require.Empty(t, gitTest(t, project, "status", "--porcelain=v1", "--untracked-files=all"))
	_, err = os.Stat(lock)
	require.True(t, os.IsNotExist(err))
}
