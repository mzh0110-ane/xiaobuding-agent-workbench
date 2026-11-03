package reliable

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadOnlyInspectionUsesExistingLinkedWorktree(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	project := fixtureRepo(t)
	linked := filepath.Join(t.TempDir(), "prepared")
	gitTest(t, project, "worktree", "add", "--detach", linked, "HEAD")
	before := gitTest(t, project, "worktree", "list", "--porcelain")
	task := &Task{ID: "inspect-linked", Project: linked, Goal: "了解当前项目", ReadOnly: true, State: stateQueued}
	require.NoError(t, e.Store.Create(task))
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	after, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateInspected, after.State)
	require.Empty(t, after.Worktree)
	require.Equal(t, before, gitTest(t, project, "worktree", "list", "--porcelain"))
	require.NoError(t, e.ArchiveTask(task.ID, true))
	require.NoError(t, e.Store.Create(&Task{ID: "next", Project: linked, Goal: "next", State: stateQueued}))
	require.Error(t, e.Accept(context.Background(), task.ID))
}

func TestInspectionRejectsMissingAndExternalCitations(t *testing.T) {
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("source"), 0600))
	for _, paths := range [][]string{nil, {"missing.txt"}, {"../private.txt"}, {".git/config"}} {
		_, err := inspectionSources(project, paths)
		require.Error(t, err)
	}
	valid, err := inspectionSources(project, []string{"app.txt"})
	require.NoError(t, err)
	require.Len(t, valid, 1)
}

func TestWorktreeRejectsOversizedIndexBeforeCopy(t *testing.T) {
	project := fixtureRepo(t)
	hash, err := localGit(context.Background(), project, []byte("source"), "hash-object", "-w", "--stdin")
	require.NoError(t, err)
	var entries strings.Builder
	for i := 0; i < 30000; i++ {
		_, err = fmt.Fprintf(&entries, "100644 %s\tbulk/file-%05d.txt\n", strings.TrimSpace(string(hash)), i)
		require.NoError(t, err)
	}
	_, err = localGit(context.Background(), project, []byte(entries.String()), "update-index", "--index-info")
	require.NoError(t, err)
	root := t.TempDir()
	_, _, err = PrepareWorktree(context.Background(), root, project, "oversized")
	require.ErrorContains(t, err, "source_snapshot_too_large")
	_, err = os.Stat(filepath.Join(root, "worktrees", "oversized"))
	require.True(t, os.IsNotExist(err))
	require.Empty(t, strings.TrimSpace(gitTest(t, project, "branch", "--list", "xiaobuding/*")))
}

func TestProcessOutputLimitCannotBeBypassedByCopy(t *testing.T) {
	cancelled := false
	buffer := &limitedBuffer{cancel: func() { cancelled = true }}
	_, err := io.Copy(buffer, io.LimitReader(strings.NewReader(strings.Repeat("x", 17<<20)), 17<<20))
	require.Error(t, err)
	require.True(t, cancelled)
	require.LessOrEqual(t, buffer.Len(), 16<<20)
}
