package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStorageCommandKeepsNativeHomesAndUsesOwnedTemporaryDirectory(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("unused")
	cmd.Env = []string{"TEMP=C:\\temp", "tmp=C:\\tmp", "GOCACHE=C:\\go-cache", "CODEX_HOME=existing-codex", "DSH_HOME=existing-harness", "HOME=existing-home"}
	cleanup, err := prepareStorageCommand(storageContext(context.Background(), root), cmd)
	require.NoError(t, err)
	env := strings.Join(cmd.Env, "\n")
	require.Contains(t, env, "CODEX_HOME=existing-codex")
	require.Contains(t, env, "DSH_HOME=existing-harness")
	require.Contains(t, env, "HOME=existing-home")
	require.NotContains(t, env, "C:\\temp")
	temp := storageEnvironmentValue(cmd.Env, "TEMP")
	require.Equal(t, filepath.Join(root, "temp"), filepath.Dir(temp))
	require.DirExists(t, temp)
	require.Equal(t, temp, storageEnvironmentValue(cmd.Env, "TMP"))
	require.Equal(t, filepath.Join(root, "cache", "go"), storageEnvironmentValue(cmd.Env, "GOCACHE"))
	cleanup()
	require.NoDirExists(t, temp)
}

func TestTemporaryCleanupPreservesActiveUnownedRecentAndProjectData(t *testing.T) {
	root := t.TempDir()
	e := &Engine{Root: root}
	old := time.Now().Add(-8 * 24 * time.Hour)
	stale := storageFixture(t, root, "process-stale", true, old)
	recent := storageFixture(t, root, "process-recent", true, time.Now())
	unowned := storageFixture(t, root, "process-user", false, old)
	cmd := exec.Command("unused")
	finish, err := prepareStorageCommand(storageContext(context.Background(), root), cmd)
	require.NoError(t, err)
	defer finish()
	active := storageEnvironmentValue(cmd.Env, "TEMP")
	require.NoError(t, os.Chtimes(filepath.Join(active, storageOwnerFile), old, old))
	require.NoError(t, os.Chtimes(active, old, old))
	protected := filepath.Join(root, "worktrees", "unaccepted.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(protected), 0700))
	require.NoError(t, os.WriteFile(protected, []byte("preserve"), 0600))
	preview, err := e.CleanupTemporary(context.Background(), true)
	require.NoError(t, err)
	require.EqualValues(t, 1, preview.Files)
	require.DirExists(t, stale)
	report, err := e.CleanupTemporary(context.Background(), false)
	require.NoError(t, err)
	require.EqualValues(t, 1, report.RemovedFiles)
	require.NoDirExists(t, stale)
	for _, path := range []string{recent, unowned, active} {
		require.DirExists(t, path)
	}
	require.FileExists(t, protected)
}

func TestStorageUsageCountsOwnedCategoriesAndHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "evidence"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "evidence", "report.txt"), []byte("proof"), 0600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "context"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "context", "retained.json"), []byte("context"), 0600))
	e := &Engine{Root: root}
	report, err := e.StorageUsage(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 12, report.TotalBytes)
	require.Len(t, report.Entries, 6)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = e.StorageUsage(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOwnedProcessStorageEnvironmentAndScratchCleanup(t *testing.T) {
	root := t.TempDir()
	executable, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(executable, "-test.run=^TestStorageEnvironmentSubprocess$")
	cmd.Env = isolatedEnvironment(os.Environ(), map[string]string{"XIAOBUDING_STORAGE_HELPER": "1", "CODEX_HOME": "native-codex", "DSH_HOME": "native-harness"}, nil)
	var output bytes.Buffer
	cmd.Stdout = &output
	require.NoError(t, runOwnedCommand(storageContext(context.Background(), root), cmd))
	var observed map[string]string
	require.NoError(t, json.Unmarshal(output.Bytes(), &observed))
	require.Equal(t, "native-codex", observed["CODEX_HOME"])
	require.Equal(t, "native-harness", observed["DSH_HOME"])
	require.Equal(t, "true", observed["exists"])
	require.Equal(t, filepath.Join(root, "temp"), filepath.Dir(observed["TEMP"]))
	require.NoDirExists(t, observed["TEMP"])
}

func TestStorageEnvironmentSubprocess(t *testing.T) {
	if os.Getenv("XIAOBUDING_STORAGE_HELPER") != "1" {
		return
	}
	_, err := os.Stat(os.Getenv("TEMP"))
	data, _ := json.Marshal(map[string]string{"TEMP": os.Getenv("TEMP"), "CODEX_HOME": os.Getenv("CODEX_HOME"), "DSH_HOME": os.Getenv("DSH_HOME"), "exists": fmt.Sprint(err == nil)})
	fmt.Print(string(data))
	os.Exit(0)
}

func TestTemporaryCleanupDoesNotFollowDirectoryLinks(t *testing.T) {
	root, external := t.TempDir(), t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "temp"), 0700))
	link := filepath.Join(root, "temp", "process-link")
	if err := os.Symlink(external, link); err != nil {
		t.Skipf("directory symlinks unavailable: %v", err)
	}
	require.NoError(t, os.WriteFile(filepath.Join(external, "keep.txt"), []byte("preserve"), 0600))
	e := &Engine{Root: root}
	_, err := e.CleanupTemporary(context.Background(), false)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(external, "keep.txt"))
}

func TestCodexStorageOverridesKeepEphemeralArgumentsAndNativeAccountHome(t *testing.T) {
	root := t.TempDir()
	runner := &CodexRunner{Home: "original-native-home", Model: "existing-model"}
	request := RoleRequest{Directory: root, ReadOnly: true}
	original := runner.arguments(request, "schema.json", "response.json")
	args, err := codexStorageArguments(storageContext(context.Background(), root), original)
	require.NoError(t, err)
	require.Equal(t, original, args[:len(original)])
	require.Contains(t, strings.Join(args, "\n"), "history.persistence=\"none\"")
	require.Contains(t, strings.Join(args, "\n"), "sqlite_home=")
	require.Contains(t, strings.Join(args, "\n"), "log_dir=")
	require.NotContains(t, strings.Join(args, "\n"), "CODEX_HOME")
	require.Equal(t, "read-only", args[5])
	require.Equal(t, "original-native-home", runner.Home)
	require.DirExists(t, filepath.Join(root, "cache", "codex", "state"))
	require.DirExists(t, filepath.Join(root, "cache", "codex", "log"))
	withoutContext, err := codexStorageArguments(context.Background(), original)
	require.NoError(t, err)
	require.Equal(t, original, withoutContext)
}

func storageFixture(t *testing.T, root, name string, owned bool, modified time.Time) string {
	t.Helper()
	dir := filepath.Join(root, "temp", name)
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.tmp"), []byte("temporary"), 0600))
	if owned {
		require.NoError(t, os.WriteFile(filepath.Join(dir, storageOwnerFile), []byte(storageOwnerVersion), 0600))
		require.NoError(t, os.Chtimes(filepath.Join(dir, storageOwnerFile), modified, modified))
	}
	require.NoError(t, os.Chtimes(filepath.Join(dir, "data.tmp"), modified, modified))
	require.NoError(t, os.Chtimes(dir, modified, modified))
	return dir
}

func storageEnvironmentValue(env []string, key string) string {
	for _, item := range env {
		pair := strings.SplitN(item, "=", 2)
		if len(pair) == 2 && strings.EqualFold(pair[0], key) {
			return pair[1]
		}
	}
	return ""
}
