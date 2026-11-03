package reliable

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func codexDiscoveryEngine(t *testing.T, runner AgentRunner) *Engine {
	t.Helper()
	root := t.TempDir()
	store, err := OpenStore(filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{Store: store, Root: root, Runner: runner}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func TestCodexReservationPinsExecutableUntilNextRun(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	modified := time.Unix(1_700_000_000, 0)
	old := codexPEFixture(t, root, codexOldFixture, modified)
	runner := &CodexRunner{Executable: old, Home: t.TempDir(), Model: "selected-model", Provider: "openai"}
	e := codexDiscoveryEngine(t, &ProviderRouter{Runners: map[string]AgentRunner{connectionCodex: runner}})
	ctx, release, err := e.reserve(context.Background(), "first-run")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	newest := codexPEFixture(t, root, codexNewFixture, modified.Add(time.Hour))
	for _, role := range []string{roleAuditor, roleImplementer, roleReviewer} {
		if path, err := runner.executableForContext(ctx); err != nil || path != old {
			t.Fatalf("%s switched executable within an active run: %q %v", role, path, err)
		}
	}
	next, nextRelease, err := e.reserve(context.Background(), "next-run")
	if err != nil {
		t.Fatal(err)
	}
	defer nextRelease()
	if path, err := runner.executableForContext(next); err != nil || path != newest {
		t.Fatalf("next run did not select the update: %q %v", path, err)
	}
	if runner.Executable != old || runner.Model != "selected-model" || runner.Provider != "openai" {
		t.Fatal("discovery mutated the shared runner or model connection")
	}
}

func TestCodexReservationDefersMissingExecutableErrorUntilCodexRuns(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	runner := &CodexRunner{Executable: filepath.Join(root, codexOldFixture, "codex.exe")}
	harness := &codexUnrelatedRunner{}
	router := &ProviderRouter{Runners: map[string]AgentRunner{connectionCodex: runner, connectionHarness: harness}}
	e := codexDiscoveryEngine(t, router)
	ctx, release, err := e.reserve(context.Background(), "harness-run")
	if err != nil {
		t.Fatalf("unused missing Codex prevented run reservation: %v", err)
	}
	defer release()
	if _, err := router.Run(ctx, RoleRequest{ProviderID: connectionHarness}); err != nil {
		t.Fatalf("unused Codex error blocked Harness: %v", err)
	}
	_ = codexPEFixture(t, root, codexNewFixture, time.Now())
	if _, err := runner.Run(ctx, RoleRequest{}); err == nil {
		t.Fatal("the active run discarded its original executable error snapshot")
	}
	next, nextRelease, err := e.reserve(context.Background(), "next-codex-run")
	if err != nil {
		t.Fatal(err)
	}
	defer nextRelease()
	if _, err := runner.executableForContext(next); err != nil {
		t.Fatalf("next run retained a previous run's discovery failure: %v", err)
	}
}

type codexUnrelatedRunner struct{}

func (*codexUnrelatedRunner) Run(context.Context, RoleRequest) (*AgentResult, error) {
	return &AgentResult{Session: "harness-only"}, nil
}

func TestCodexExecutableSnapshotsAreImmutableUnderConcurrentReads(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	modified := time.Unix(1_700_000_000, 0)
	old := codexPEFixture(t, root, codexOldFixture, modified)
	runner := &CodexRunner{Executable: old}
	e := codexDiscoveryEngine(t, runner)
	ctx, release, err := e.reserve(context.Background(), "active")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	newest := codexPEFixture(t, root, codexNewFixture, modified.Add(time.Hour))
	errors := make(chan error, 32)
	var workers sync.WaitGroup
	for i := range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if path, err := runner.executableForContext(ctx); err != nil || path != old {
				errors <- fmt.Errorf("active snapshot changed: %q %w", path, err)
			}
			next, done, err := e.reserve(context.Background(), fmt.Sprintf("next-%d", i))
			if err != nil {
				errors <- err
				return
			}
			defer done()
			if path, err := runner.executableForContext(next); err != nil || path != newest {
				errors <- fmt.Errorf("next snapshot missed update: %q %w", path, err)
			}
		}()
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	if runner.Executable != old {
		t.Fatal("concurrent discovery changed the shared runner")
	}
}

func TestCodexDiscoveryRejectsSymbolicLinks(t *testing.T) {
	for _, linkFile := range []bool{false, true} {
		t.Run(fmt.Sprintf("file-%t", linkFile), func(t *testing.T) {
			root := codexManagedFixtureRoot(t)
			valid := codexPEFixture(t, root, codexOldFixture, time.Unix(1_700_000_000, 0))
			external := codexPEFixture(t, t.TempDir(), codexNewFixture, time.Now())
			link := filepath.Join(root, codexNewFixture)
			target := filepath.Dir(external)
			if linkFile {
				if err := os.Mkdir(link, 0700); err != nil {
					t.Fatal(err)
				}
				link = filepath.Join(link, "codex.exe")
				target = external
			}
			if err := os.Symlink(target, link); err != nil {
				t.Skipf("symbolic link creation unavailable: %v", err)
			}
			if path, err := resolveCodexExecutable(valid); err != nil || path != valid {
				t.Fatalf("linked candidate was accepted: %q %v", path, err)
			}
		})
	}
}
