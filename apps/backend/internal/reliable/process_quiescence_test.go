package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOwnedCleanupCancellationRetainsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done, ready := startQuiescenceHelper(t, ctx, "hold")
	require.Eventually(t, func() bool { _, err := os.Stat(ready); return err == nil }, 10*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(10 * time.Second):
		t.Fatal("owned process did not stop after cancellation")
	}
}

func TestOwnedCleanupProvesWindowsDescendantsStopped(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows permits waiting on an owned descendant handle")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, proof := observeOwnedCleanup(ctx)
	done, ready := startQuiescenceHelper(t, ctx, "tree")
	var childPID int
	require.Eventually(t, func() bool {
		raw, err := os.ReadFile(ready)
		if err == nil {
			childPID, err = strconv.Atoi(strings.TrimSpace(string(raw)))
		}
		return err == nil && childPID > 0
	}, 10*time.Second, 10*time.Millisecond)
	child, err := os.FindProcess(childPID)
	require.NoError(t, err)
	childStopped := make(chan error, 1)
	go func() { _, waitErr := child.Wait(); childStopped <- waitErr }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-childStopped:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			_ = child.Kill()
			t.Error("owned descendant did not exit")
		}
	})
	require.False(t, proof.verified(), "live owned work must not satisfy the recovery guard")
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.True(t, proof.verified(), "the owned job must be confirmed empty before return")
}

func TestOwnedCleanupProofRetainsFailureAcrossLaterSuccess(t *testing.T) {
	ctx, proof := observeOwnedCleanup(context.Background())
	require.True(t, proof.verified(), "a fixture without subprocesses is already quiescent")
	finish := trackOwnedCleanup(ctx)
	require.False(t, proof.verified())
	finish(errors.New("job query failed"))
	trackOwnedCleanup(ctx)(nil)
	require.False(t, proof.verified(), "a later successful command cannot launder a failed cleanup")
	engine, task := quiescenceFixture(t)
	require.ErrorContains(t, engine.recordProcessQuiescence(task, proof), "recovery_process_unknown")
	require.ErrorContains(t, engine.processQuiescent(task), "recovery_process_unknown")
}

func TestProcessQuiescenceRequiresMatchingHistory(t *testing.T) {
	engine, task := quiescenceFixture(t)
	require.ErrorContains(t, engine.processQuiescent(task), "recovery_process_unknown")
	_, proof := observeOwnedCleanup(context.Background())
	require.NoError(t, engine.recordProcessQuiescence(task, proof))
	require.NoError(t, engine.processQuiescent(task))
	task.Runs = append(task.Runs, Dispatch{ID: "second", Status: "completed"})
	require.ErrorContains(t, engine.processQuiescent(task), "recovery_process_unknown")
	task.Runs = task.Runs[:1]
	task.Base = "other-base"
	require.ErrorContains(t, engine.processQuiescent(task), "recovery_process_unknown")
}

func TestProcessQuiescencePreservesImmutableReceipt(t *testing.T) {
	engine, task := quiescenceFixture(t)
	_, proof := observeOwnedCleanup(context.Background())
	require.NoError(t, engine.recordProcessQuiescence(task, proof))
	name, _, err := quiescenceReceipt(task)
	require.NoError(t, err)
	path := filepath.Join(engine.Root, name)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, engine.recordProcessQuiescence(task, proof))
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, raw, unchanged)
	require.NoError(t, os.WriteFile(path, []byte(`{"task_id":"other"}`), 0600))
	require.Error(t, engine.recordProcessQuiescence(task, proof))
	require.ErrorContains(t, engine.processQuiescent(task), "recovery_process_unknown")
}

func TestProcessQuiescenceAcceptsExplicitWindowsServiceExit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("service exit receipts are specific to Windows Job Objects")
	}
	engine, task := quiescenceFixture(t)
	name, receipt, err := quiescenceReceipt(task)
	require.NoError(t, err)
	receipt.Method = "windows-service-exit"
	raw, err := json.Marshal(receipt)
	require.NoError(t, err)
	path := filepath.Join(engine.Root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, raw, 0600))
	require.NoError(t, engine.processQuiescent(task))
	_, proof := observeOwnedCleanup(context.Background())
	require.NoError(t, engine.recordProcessQuiescence(task, proof), "an existing exit receipt remains immutable")
}

func TestProcessQuiescenceDoesNotDemandProofForUnstartedWork(t *testing.T) {
	engine, task := quiescenceFixture(t)
	task.Runs = []Dispatch{{ID: "unstarted", Status: dispatchNotStarted}}
	require.NoError(t, engine.processQuiescent(task))
	require.NoError(t, engine.recordProcessQuiescence(task, nil))
	_, err := os.Stat(filepath.Join(engine.Root, "quiescence"))
	require.True(t, os.IsNotExist(err), "a passive guard must not create bookkeeping")
}

func TestFailedCleanupInvalidatesEarlierReceiptWithoutNativeRuns(t *testing.T) {
	engine, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(func() { require.NoError(t, engine.Close()) })
	oldName, oldReceipt, err := quiescenceReceipt(task)
	require.NoError(t, err)
	oldPath := filepath.Join(engine.Root, oldName)
	require.NoError(t, os.MkdirAll(filepath.Dir(oldPath), 0700))
	require.NoError(t, exclusiveJSON(oldPath, oldReceipt))
	oldBytes, err := os.ReadFile(oldPath)
	require.NoError(t, err)
	ctx, proof := observeOwnedCleanup(context.Background())
	trackOwnedCleanup(ctx)(errors.New("owned Git job cleanup failed"))
	require.ErrorContains(t, engine.recordProcessQuiescence(task, proof), "recovery_process_unknown")
	saved, err := engine.Store.Get(task.ID)
	require.NoError(t, err)
	raw, err := json.Marshal(saved)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	require.NotEmpty(t, fields["process_cleanup_epoch"], "cleanup failure must survive reason replacement and restart")
	require.ErrorContains(t, engine.processQuiescent(saved), "recovery_process_unknown")
	newName, _, err := quiescenceReceipt(saved)
	require.NoError(t, err)
	require.NotEqual(t, oldName, newName, "an earlier receipt cannot prove cleanup of this execution")
	unchanged, err := os.ReadFile(oldPath)
	require.NoError(t, err)
	require.Equal(t, oldBytes, unchanged)
}

func TestOwnedCleanupPreservesExitFailureWithoutLosingCleanupProof(t *testing.T) {
	ctx, proof := observeOwnedCleanup(context.Background())
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOwnedCleanupHelper$")
	ready := filepath.Join(t.TempDir(), "ready")
	cmd.Env = append(os.Environ(), "XBD_OWNED_CLEANUP_HELPER=fail", "XBD_OWNED_CLEANUP_READY="+ready)
	var exitErr *exec.ExitError
	require.ErrorAs(t, runOwnedCommand(ctx, cmd), &exitErr)
	require.Equal(t, 3, exitErr.ExitCode())
	require.Equal(t, runtime.GOOS == "windows", proof.verified())
}

func quiescenceFixture(t *testing.T) (*Engine, *Task) {
	t.Helper()
	return &Engine{Root: t.TempDir()}, &Task{ID: "owned-task", Base: "base", Runs: []Dispatch{{ID: "first", Status: "failed", ProviderID: "codex"}}}
}

func startQuiescenceHelper(t *testing.T, ctx context.Context, mode string) (<-chan error, string) {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	ready := filepath.Join(t.TempDir(), "ready")
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOwnedCleanupHelper$")
	cmd.Env = append(os.Environ(), "XBD_OWNED_CLEANUP_HELPER="+mode, "XBD_OWNED_CLEANUP_READY="+ready)
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() { defer close(finished); done <- runOwnedCommand(ctx, cmd) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(10 * time.Second):
			t.Error("owned helper did not finish cleanup")
		}
	})
	return done, ready
}

func TestOwnedCleanupHelper(t *testing.T) {
	mode := os.Getenv("XBD_OWNED_CLEANUP_HELPER")
	if mode == "" {
		return
	}
	ready := os.Getenv("XBD_OWNED_CLEANUP_READY")
	data := []byte("ready")
	if mode == "tree" {
		child := exec.Command(os.Args[0], "-test.run=^TestOwnedCleanupHelper$")
		child.Env = append(os.Environ(), "XBD_OWNED_CLEANUP_HELPER=child", "XBD_OWNED_CLEANUP_READY="+ready+".child")
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		data = []byte(strconv.Itoa(child.Process.Pid))
	}
	if err := os.WriteFile(ready, data, 0600); err != nil {
		os.Exit(2)
	}
	if mode == "fail" {
		os.Exit(3)
	}
	for {
		time.Sleep(time.Hour)
	}
}
