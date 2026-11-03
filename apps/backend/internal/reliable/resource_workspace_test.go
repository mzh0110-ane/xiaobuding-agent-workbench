package reliable

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func resourceFixture(t *testing.T) string {
	t.Helper()
	dir := fixtureRepo(t)
	ctx := context.Background()
	blob, err := localGit(ctx, dir, []byte("image\x00input"), "hash-object", "-w", "--stdin")
	require.NoError(t, err)
	var index, skipped strings.Builder
	for i := 0; i < 30001; i++ {
		name := fmt.Sprintf("static/figs/%05d.wmf", i)
		fmt.Fprintf(&index, "100644 %s\t%s\n", strings.TrimSpace(string(blob)), name)
		skipped.WriteString(name + "\x00")
	}
	_, err = localGit(ctx, dir, []byte(index.String()), "update-index", "--index-info")
	require.NoError(t, err)
	gitTest(t, dir, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "resource fixture")
	_, err = localGit(ctx, dir, []byte(skipped.String()), "update-index", "--skip-worktree", "-z", "--stdin")
	require.NoError(t, err)
	require.Empty(t, gitTest(t, dir, "status", "--porcelain"))
	return dir
}

// @covers AC-RELIABLE-RESOURCES-001.1, AC-RELIABLE-RESOURCES-001.2
func TestLargeResourcePreparation(t *testing.T) {
	project := resourceFixture(t)
	before := gitTest(t, project, "ls-files", "-v", "--stage")
	ctx := context.Background()
	work, base, err := PrepareWorktree(ctx, t.TempDir(), project, "resources")
	require.NoError(t, err, "binary input libraries must not trip the mutable-source ceiling")
	require.NotEmpty(t, base)
	raw, err := os.ReadFile(filepath.Join(work, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, "before\n", string(raw))
	_, err = os.Stat(filepath.Join(work, "static", "figs", "00000.wmf"))
	require.True(t, os.IsNotExist(err), "input resources must remain deferred")
	snap, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	require.Len(t, snap.Files, 1)
	require.Equal(t, before, gitTest(t, project, "ls-files", "-v", "--stage"))
	require.Empty(t, gitTest(t, project, "status", "--porcelain"))
}

func TestSparsePreparationRecovery(t *testing.T) {
	project := resourceFixture(t)
	root := t.TempDir()
	ctx := context.Background()
	work, base, err := PrepareWorktree(ctx, root, project, "recover-resource")
	require.NoError(t, err)
	markerPath, err := profileMarkerPath(ctx, work)
	require.NoError(t, err)
	require.NoError(t, os.Remove(markerPath+".ready"))
	again, againBase, err := PrepareWorktree(ctx, root, project, "recover-resource")
	require.NoError(t, err)
	require.Equal(t, work, again)
	require.Equal(t, base, againBase)
	require.NoError(t, os.WriteFile(filepath.Join(work, "app.txt"), []byte("preserve user work"), 0600))
	_, _, err = PrepareWorktree(ctx, root, project, "recover-resource")
	require.Error(t, err)
	raw, err := os.ReadFile(filepath.Join(work, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, "preserve user work", string(raw))
}

func TestResourceFiltersRejectedBeforeCopy(t *testing.T) {
	project := resourceFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(project, ".gitattributes"), []byte("*.wmf filter=external-transform\n"), 0600))
	gitTest(t, project, "add", "--", ".gitattributes")
	gitTest(t, project, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "resource transformation")
	_, _, err := PrepareWorktree(context.Background(), t.TempDir(), project, "filtered-resource")
	require.Error(t, err, "transformed input bytes cannot be treated as raw frozen blob contents")
}

func TestQuestionBankDataDeferredButConfigEditable(t *testing.T) {
	project := resourceFixture(t)
	require.NoError(t, os.MkdirAll(filepath.Join(project, "artifacts"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(project, "artifacts", "question-candidates.json"), []byte(strings.Repeat(" ", 17<<20)), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(project, "package.json"), []byte(`{"name":"source-config"}`), 0600))
	gitTest(t, project, "add", "--", "artifacts/question-candidates.json", "package.json")
	gitTest(t, project, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "data and configuration")
	work, _, err := PrepareWorktree(context.Background(), t.TempDir(), project, "json-input")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(work, "artifacts", "question-candidates.json"))
	require.True(t, os.IsNotExist(err), "question data must not be copied as executable source")
	raw, err := os.ReadFile(filepath.Join(work, "package.json"))
	require.NoError(t, err)
	require.Contains(t, string(raw), "source-config")
	_, err = TakeSnapshot(context.Background(), work)
	require.NoError(t, err)
}

func TestResourceSnapshotHonorsIgnoredOutputs(t *testing.T) {
	project := resourceFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(project, ".gitignore"), []byte("*.log\n"), 0600))
	gitTest(t, project, "add", "--", ".gitignore")
	gitTest(t, project, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "ignore runtime logs")
	ctx := context.Background()
	work, _, err := PrepareWorktree(ctx, t.TempDir(), project, "ignore-outputs")
	require.NoError(t, err)
	before, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(work, "runtime.log"), []byte("generated check output"), 0600))
	after, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	require.Equal(t, before.Hash, after.Hash, "ignored runtime output is not mutable project source")
}

func TestExistingResourceProjectPreparation(t *testing.T) {
	project := os.Getenv("XIAOBUDING_RESOURCE_PROJECT")
	if project == "" {
		t.Skip("explicit resource-project preparation is disabled")
	}
	root := os.Getenv("XIAOBUDING_RESOURCE_VALIDATION_ROOT")
	require.NotEmpty(t, root)
	ctx := context.Background()
	before := gitTest(t, project, "ls-files", "-v", "--stage")
	id := os.Getenv("XIAOBUDING_RESOURCE_VALIDATION_ID")
	if id == "" {
		id = "resource-admission-check"
	}
	work, _, err := PrepareWorktree(ctx, root, project, id)
	require.NoError(t, err)
	snap, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	require.NotEmpty(t, snap.ResourceHash)
	m, _, err := loadResourceProfile(ctx, work)
	require.NoError(t, err)
	for path := range m.Resources {
		if strings.HasSuffix(path, ".wmf") {
			require.NoError(t, materializeResources(ctx, work, []string{path, "data/questions_phy.json"}))
			break
		}
	}
	after, err := TakeSnapshot(ctx, work)
	require.NoError(t, err)
	require.Equal(t, snap.Hash, after.Hash)
	require.NotEmpty(t, after.AvailableResources)
	require.Equal(t, before, gitTest(t, project, "ls-files", "-v", "--stage"))
	require.Empty(t, gitTest(t, project, "status", "--porcelain"))
	t.Logf("prepared %d source entries; worktree %s; resource manifest %s", len(snap.Files), work, snap.ResourceHash)
}
