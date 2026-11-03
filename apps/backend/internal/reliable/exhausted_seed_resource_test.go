package reliable

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func smallSeedResourceParent(t *testing.T, agent AgentRunner) (*Engine, *Task, *resourceManifest) {
	t.Helper()
	e, parent := fixtureEngine(t, agent, passingCheck)
	t.Cleanup(func() { _ = e.Close() })
	if err := os.MkdirAll(filepath.Join(parent.Project, "static", "figs"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"00000.wmf", "00001.wmf"} {
		mustRecoveryWrite(t, filepath.Join(parent.Project, "static", "figs", name), "frozen input "+name)
	}
	gitTest(t, parent.Project, "add", "--", ".")
	gitTest(t, parent.Project, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "frozen inputs")
	m := &resourceManifest{Source: map[string]string{"app.txt": strings.TrimSpace(gitTest(t, parent.Project, "rev-parse", "HEAD:app.txt"))}, Resources: map[string]string{}}
	for _, name := range []string{"static/figs/00000.wmf", "static/figs/00001.wmf"} {
		m.Resources[name] = strings.TrimSpace(gitTest(t, parent.Project, "rev-parse", "HEAD:"+name))
	}
	if err := completeResourceInventory(context.Background(), parent.Project, m); err != nil {
		t.Fatal(err)
	}
	if err := resourceAdmissionPolicy(context.Background(), parent.Project, m); err != nil {
		t.Fatal(err)
	}
	parent = prepareSmallResourceCandidate(t, e, parent, m)
	parent, err := e.Store.Update(parent.ID, func(saved *Task) error {
		saved.State, saved.Round, saved.ContinuePending = stateExhausted, MaxRounds, false
		saved.ApprovedPaths = []string{"app.txt"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return e, parent, m
}

func prepareSmallResourceCandidate(t *testing.T, e *Engine, task *Task, m *resourceManifest) *Task {
	t.Helper()
	worktree, base, err := prepareResourceWorktree(context.Background(), e.Root, task.Project, task.ID, filepath.Join(e.Root, "worktrees", task.ID), m.Base, m)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := TakeSnapshot(context.Background(), worktree)
	if err != nil {
		t.Fatal(err)
	}
	task, err = e.Store.Update(task.ID, func(saved *Task) error {
		saved.Worktree, saved.Base, saved.Baseline = worktree, base, baseline
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.saveJSON(task.ID, "contract.json", task); err != nil {
		t.Fatal(err)
	}
	return task
}

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedBatchPreservesFrozenResourcesAndChildGuard(t *testing.T) {
	agent := &fixtureAgent{}
	e, parent, manifest := smallSeedResourceParent(t, agent)
	ctx := context.Background()
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "app.txt"), "retained source\n")
	input := "static/figs/00000.wmf"
	if err := materializeResources(ctx, parent.Worktree, []string{input}); err != nil {
		t.Fatal(err)
	}
	child := claimExhaustedChild(t, e, parent, "resource-child")
	source, err := TakeSnapshot(ctx, parent.Worktree)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.ensureExhaustedIntent(child, child.SeedPending, source); err != nil {
		t.Fatal(err)
	}
	if err = e.ensureSeedModelPlan(child, child.SeedPending); err != nil {
		t.Fatal(err)
	}
	child = prepareSmallResourceCandidate(t, e, child, manifest)
	if err = e.prepareExhaustedSeed(ctx, child); err != nil {
		t.Fatal(err)
	}
	snap, err := TakeSnapshot(ctx, child.Worktree)
	if err != nil || snap.ResourceHash != parent.Baseline.ResourceHash || len(snap.AvailableResources) != 1 || snap.AvailableResources[input] == "" {
		t.Fatalf("frozen resources lost during continuation: snapshot=%+v error=%v", snap, err)
	}
	if err = materializeResources(ctx, parent.Worktree, []string{"static/figs/00001.wmf"}); err != nil {
		t.Fatal(err)
	}
	if err = e.prepareExhaustedSeed(ctx, child); err != nil {
		t.Fatalf("additional frozen reads invalidated retained source: %v", err)
	}
	if err = os.WriteFile(filepath.Join(child.Worktree, filepath.FromSlash(input)), []byte("changed frozen input"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Store.Update(child.ID, func(saved *Task) error {
		saved.State, saved.ContinuePending = stateInterrupted, false
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err = e.RunTask(ctx, child.ID)
	if err == nil || !strings.Contains(err.Error(), "recovery_resource_invalid") && !strings.Contains(err.Error(), "recovery_candidate_invalid") || agent.total != 0 {
		t.Fatalf("changed inherited frozen input reached an agent: calls=%d error=%v", agent.total, err)
	}
}
