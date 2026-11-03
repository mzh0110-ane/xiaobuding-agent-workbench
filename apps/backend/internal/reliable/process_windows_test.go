package reliable

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestWindowsCancellationTerminatesOwnedDescendants(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "parent.cjs")
	marker := filepath.Join(dir, "child.pid")
	code := `const {spawn}=require('node:child_process');const fs=require('node:fs');const c=spawn(process.execPath,['-e','setInterval(()=>{},1000)'],{stdio:'inherit'});fs.writeFileSync(process.argv[2],String(c.pid));setInterval(()=>{},1000);`
	require.NoError(t, os.WriteFile(script, []byte(code), 0600))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var runErr error
	go func() { _, _, _, runErr = runProcess(ctx, dir, os.Environ(), "", "node", script, marker); close(done) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("process runner did not stop")
		}
	})
	var childPID int
	require.Eventually(t, func() bool {
		raw, err := os.ReadFile(marker)
		if err != nil {
			return false
		}
		childPID, err = strconv.Atoi(strings.TrimSpace(string(raw)))
		return err == nil && childPID > 0
	}, 10*time.Second, 20*time.Millisecond)
	child, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(childPID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = windows.CloseHandle(child) })
	state, err := windows.WaitForSingleObject(child, 0)
	require.NoError(t, err)
	require.Equal(t, uint32(windows.WAIT_TIMEOUT), state, "positive control: child must be alive before cancellation")
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("cancellation left a runner waiting on child pipes")
	}
	require.Error(t, runErr)
	state, err = windows.WaitForSingleObject(child, 5000)
	require.NoError(t, err)
	require.Equal(t, uint32(windows.WAIT_OBJECT_0), state, "owned child survived cancellation")
}
