package reliable

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceMaterialization(t *testing.T) {
	ctx := context.Background()
	project := resourceFixture(t)
	work, base, err := PrepareWorktree(ctx, t.TempDir(), project, "materialize")
	require.NoError(t, err)
	before, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	path := "static/figs/00000.wmf"
	require.NoError(t, materializeResources(ctx, work, []string{path}))
	after, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	require.Equal(t, before.Hash, after.Hash)
	require.NotEmpty(t, after.AvailableResources[path])
	require.NoError(t, os.WriteFile(filepath.Join(work, "app.txt"), []byte("reviewed\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(work, "static", "figs", "new-diagram.png"), []byte("new candidate image\x00"), 0600))
	patch := filepath.Join(t.TempDir(), "change.patch")
	require.NoError(t, ExportPatch(ctx, work, base, patch))
	raw, err := os.ReadFile(patch)
	require.NoError(t, err)
	require.NotContains(t, string(raw), ".wmf")
	require.Contains(t, string(raw), "new-diagram.png", "new generated assets must remain ordinary candidate changes")
	require.NoError(t, ApplyPatch(ctx, project, base, patch))
	raw, err = os.ReadFile(filepath.Join(project, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, "reviewed\n", string(raw))
}

func TestResourceDestinationGuard(t *testing.T) {
	ctx := context.Background()
	work, _, err := PrepareWorktree(ctx, t.TempDir(), resourceFixture(t), "destination")
	require.NoError(t, err)
	for _, path := range []string{"../app.txt", "static/figs/00000.WMF", "missing.png", ".git/config"} {
		require.Error(t, materializeResources(ctx, work, []string{path}))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(work, "static", "figs"), 0700))
	path := filepath.Join(work, "static", "figs", "00000.wmf")
	require.NoError(t, os.WriteFile(path, []byte("keep changed input"), 0600))
	require.Error(t, materializeResources(ctx, work, []string{"static/figs/00000.wmf"}))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "keep changed input", string(raw))
}

func TestMaterializationCancellation(t *testing.T) {
	work, _, err := PrepareWorktree(context.Background(), t.TempDir(), resourceFixture(t), "cancel-resource")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, materializeResources(ctx, work, []string{"static/figs/00000.wmf"}))
	_, err = os.Stat(filepath.Join(work, "static", "figs", "00000.wmf"))
	require.True(t, os.IsNotExist(err))
}

func TestResourceDriftBlocksAcceptance(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	project := resourceFixture(t)
	_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.Project = project; return nil })
	require.NoError(t, err)
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	saved, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateAwaitingAcceptance, saved.State)
	require.NoError(t, materializeResources(context.Background(), saved.Worktree, []string{"static/figs/00000.wmf"}))
	require.NoError(t, os.WriteFile(filepath.Join(saved.Worktree, "static", "figs", "00000.wmf"), []byte("changed"), 0600))
	require.Error(t, e.Accept(context.Background(), task.ID))
	original, err := os.ReadFile(filepath.Join(project, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, "before\n", string(original))
}
