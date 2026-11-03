package reliable

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (e *Engine) recordLocalInitIntent(ctx context.Context, t *Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.LocalInitEvidence != "" {
		return nil
	}
	if _, err := os.Lstat(filepath.Join(t.Project, ".git")); !os.IsNotExist(err) {
		return nil
	}
	if _, err := localGit(ctx, t.Project, nil, "rev-parse", "--show-toplevel"); err == nil {
		return nil
	}
	name := "local-init-intent-" + randomID() + ".json"
	if err := e.saveJSON(t.ID, name, map[string]string{"project": t.Project}); err != nil {
		return err
	}
	saved, err := e.Store.Update(t.ID, func(saved *Task) error {
		if saved.State == stateCancelled || saved.State == stateCancelling {
			return context.Canceled
		}
		saved.LocalInitEvidence = name
		saved.SetupEvidence = append(saved.SetupEvidence, name)
		return nil
	})
	if err == nil {
		t.LocalInitEvidence, t.SetupEvidence = saved.LocalInitEvidence, saved.SetupEvidence
	}
	return err
}

func (e *Engine) ownsLocalInitialization(t *Task) (bool, error) {
	if t.LocalInitEvidence == "" {
		return false, nil
	}
	if !safeRelative(t.LocalInitEvidence) || strings.Contains(t.LocalInitEvidence, "/") {
		return false, errors.New("local initialization evidence path is invalid")
	}
	raw, err := os.ReadFile(filepath.Join(e.Root, "evidence", t.ID, t.LocalInitEvidence))
	if err != nil {
		return false, err
	}
	var intent map[string]string
	if decodeStrict(raw, &intent) != nil || projectKey(intent["project"]) != projectKey(t.Project) || !containsPath(t.SetupEvidence, t.LocalInitEvidence) {
		return false, errors.New("local initialization evidence does not match this original project")
	}
	return true, nil
}

func recoverLocalInitialization(ctx context.Context, project string) error {
	info, err := os.Lstat(filepath.Join(project, ".git"))
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("local_snapshot_incomplete: initialization metadata changed")
	}
	if _, err = os.Stat(filepath.Join(project, ".git", "index")); !os.IsNotExist(err) {
		return errors.New("local_snapshot_staged: an interrupted initialization has an unowned index; it was preserved")
	}
	refs, refsErr := localGit(ctx, project, nil, "for-each-ref", "--format=%(refname)")
	remotes, remoteErr := localGit(ctx, project, nil, "remote")
	if refsErr != nil || remoteErr != nil || len(refs) > 0 || len(remotes) > 0 {
		return errors.New("local_snapshot_incomplete: initialization acquired external history or remotes")
	}
	_, err = localGit(ctx, project, nil, "config", "--local", managedRootKey, project)
	return err
}
