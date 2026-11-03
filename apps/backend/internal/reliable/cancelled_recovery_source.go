package reliable

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

func (e *Engine) validateRecoveryCandidate(ctx context.Context, t *Task) (*Snapshot, error) {
	if t.Baseline == nil || t.Base == "" || t.Baseline.Head != t.Base {
		return nil, errors.New("recovery_candidate_invalid: initial source baseline is missing or changed")
	}
	if err := e.recoveryOwnership(ctx, t); err != nil {
		return nil, fmt.Errorf("recovery_candidate_invalid: %w", err)
	}
	var contract Task
	if err := e.readRecoveryJSON(t.ID, "contract.json", &contract); err != nil || !sameRecoveryContract(&contract, t) {
		return nil, errors.New("recovery_evidence_invalid: frozen project contract is missing or inconsistent")
	}
	if err := recoveryOriginal(ctx, t); err != nil {
		return nil, fmt.Errorf("recovery_original_changed: %w", err)
	}
	if err := e.recoveryResources(ctx, t); err != nil {
		return nil, fmt.Errorf("recovery_resource_invalid: %w", err)
	}
	snap, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return nil, fmt.Errorf("recovery_candidate_invalid: %w", err)
	}
	if snap.Head != t.Base || snap.ResourceHash != t.Baseline.ResourceHash {
		return nil, errors.New("recovery_candidate_invalid: retained source identity changed")
	}
	if err = recoverySourceScope(t, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func recoverySourceScope(t *Task, snap *Snapshot) error {
	paths := changedFiles(t.Baseline, snap)
	if err := checkScope(paths, t.ApprovedPaths); err != nil && len(paths) > 0 {
		return fmt.Errorf("recovery_scope: %w", err)
	}
	if len(t.AllowedPaths) > 0 {
		if err := checkScope(paths, t.AllowedPaths); err != nil {
			return fmt.Errorf("recovery_scope: %w", err)
		}
	}
	return nil
}

func sameRecoveryContract(c, t *Task) bool {
	return c.ID == t.ID && projectKey(c.Project) == projectKey(t.Project) && projectKey(c.Worktree) == projectKey(t.Worktree) && c.Base == t.Base && c.Goal == t.Goal &&
		reflect.DeepEqual(c.Baseline, t.Baseline) && reflect.DeepEqual(c.Commands, t.Commands) && reflect.DeepEqual(c.AllowedPaths, t.AllowedPaths) && reflect.DeepEqual(c.Attachments, t.Attachments) &&
		c.AutoChecks == t.AutoChecks && c.AutoSnapshot == t.AutoSnapshot && c.ModelPlan.Mode == t.ModelPlan.Mode &&
		c.ModelPlan.FrontendPaths == t.ModelPlan.FrontendPaths && c.ModelPlan.BackendPaths == t.ModelPlan.BackendPaths
}

func (e *Engine) recoveryOwnership(ctx context.Context, t *Task) error {
	if err := e.recoveryCandidatePath(t); err != nil {
		return err
	}
	common, err := git(ctx, t.Worktree, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	original, err := git(ctx, t.Project, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	branch, err := git(ctx, t.Worktree, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return err
	}
	head, err := git(ctx, t.Worktree, "rev-parse", "HEAD")
	if err != nil || projectKey(strings.TrimSpace(string(common))) != projectKey(strings.TrimSpace(string(original))) || strings.TrimSpace(string(branch)) != "xiaobuding/"+t.ID || strings.TrimSpace(string(head)) != t.Base {
		return errors.New("candidate branch, Git history or original repository does not match")
	}
	return nil
}

func (e *Engine) recoveryCandidatePath(t *Task) error {
	if !safeRelative(t.ID) || strings.Contains(t.ID, "/") {
		return errors.New("invalid candidate task identity")
	}
	root, rootErr := filepath.EvalSymlinks(e.Root)
	resolved, err := filepath.EvalSymlinks(t.Worktree)
	expected := filepath.Join(root, "worktrees", t.ID)
	if err != nil || rootErr != nil || !sameRecoveryPath(resolved, expected) {
		return errors.New("candidate path is missing or outside its task-owned directory")
	}
	for current := expected; !sameRecoveryPath(current, root); current = filepath.Dir(current) {
		info, statErr := os.Lstat(current)
		if statErr != nil || !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return errors.New("candidate path contains a link or substituted directory")
		}
	}
	return nil
}

func sameRecoveryPath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	return a == b || (runtime.GOOS == "windows" && strings.EqualFold(a, b))
}

func recoveryOriginal(ctx context.Context, t *Task) error {
	if err := requireClean(ctx, t.Project, t.Base); err != nil {
		return err
	}
	var total int64
	for name, want := range t.Baseline.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !safeRelative(name) {
			return errors.New("unsafe frozen source path")
		}
		got, err := fileFingerprint(t.Project, name, &total)
		if err != nil || got != want {
			return fmt.Errorf("original source differs from its frozen baseline: %s", name)
		}
	}
	return nil
}

func (e *Engine) recoverPreparation(ctx context.Context, t *Task) error {
	if t.Round != 0 || t.Baseline != nil {
		return errors.New("recovery_candidate_invalid: established candidate is missing")
	}
	for _, name := range t.SetupEvidence {
		if !strings.HasPrefix(name, "project-preparation-") {
			continue
		}
		var point localRecoveryPoint
		if err := e.readRecoveryJSON(t.ID, name, &point); err != nil || projectKey(point.Project) != projectKey(t.Project) || point.Head == "" {
			return errors.New("recovery_evidence_invalid: original preparation receipt changed")
		}
		if err := requireClean(ctx, t.Project, point.Head); err != nil {
			return fmt.Errorf("recovery_original_changed: %w", err)
		}
	}
	if t.Base != "" {
		if err := requireClean(ctx, t.Project, t.Base); err != nil {
			return fmt.Errorf("recovery_original_changed: %w", err)
		}
	}
	if _, err := os.Lstat(filepath.Join(e.Root, "worktrees", t.ID)); err == nil && len(t.SetupEvidence) == 0 {
		return errors.New("recovery_candidate_invalid: incomplete checkout has no preparation ownership receipt")
	}
	if err := e.prepare(ctx, t); err != nil {
		return fmt.Errorf("recovery_candidate_invalid: %w", err)
	}
	return nil
}
