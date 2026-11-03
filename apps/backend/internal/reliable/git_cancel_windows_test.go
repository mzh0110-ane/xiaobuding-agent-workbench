package reliable

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestGitCancellationStopsDescendantsAndReturns(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node runtime unavailable")
	}
	dir := t.TempDir()
	script, marker := filepath.Join(dir, "wait.cjs"), filepath.Join(dir, "owned.json")
	code := `const fs=require('node:fs');const {spawn}=require('node:child_process');const child=spawn(process.execPath,['-e','setInterval(()=>{},1000)'],{stdio:'inherit'});fs.writeFileSync(process.argv[2],JSON.stringify({parent:process.pid,child:child.pid}));setInterval(()=>{},1000);`
	require.NoError(t, os.WriteFile(script, []byte(code), 0600))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	var owned struct{ Parent, Child uint32 }
	needsCleanup := true
	t.Cleanup(func() {
		cancel()
		if needsCleanup && owned.Parent != 0 {
			_ = exec.Command("taskkill.exe", "/PID", strconv.FormatUint(uint64(owned.Parent), 10), "/T", "/F").Run()
		}
	})
	alias := "!node '" + filepath.ToSlash(script) + "' '" + filepath.ToSlash(marker) + "'"
	project := fixtureRepo(t)
	go func() { _, err := git(ctx, project, "-c", "alias.xbd-wait="+alias, "xbd-wait"); done <- err }()
	require.Eventually(t, func() bool {
		raw, err := os.ReadFile(marker)
		return err == nil && json.Unmarshal(raw, &owned) == nil && owned.Child > 0
	}, 5*time.Second, 10*time.Millisecond)
	child, err := windows.OpenProcess(windows.SYNCHRONIZE, false, owned.Child)
	require.NoError(t, err)
	t.Cleanup(func() { _ = windows.CloseHandle(child) })
	state, err := windows.WaitForSingleObject(child, 0)
	require.NoError(t, err)
	require.Equal(t, uint32(windows.WAIT_TIMEOUT), state)
	cancel()
	select {
	case runErr := <-done:
		require.Error(t, runErr)
	case <-time.After(4 * time.Second):
		t.Fatal("cancelled Git is still waiting on inherited pipes")
	}
	state, err = windows.WaitForSingleObject(child, 1000)
	require.NoError(t, err)
	require.Equal(t, uint32(windows.WAIT_OBJECT_0), state, "Git child survived cancellation")
	needsCleanup = false
}
