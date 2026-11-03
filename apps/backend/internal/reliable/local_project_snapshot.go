package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type pendingLocalSnapshot struct {
	Project string `json:"project"`
	Base    string `json:"base"`
	Index   string `json:"index"`
}

func localIndexHash(project string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(project, ".git", "index"))
	if os.IsNotExist(err) {
		return "missing", nil
	}
	if err != nil {
		return "", err
	}
	return digest(raw), nil
}

func ownsPendingIndex(project, head string) bool {
	raw, err := os.ReadFile(filepath.Join(project, ".git", "xiaobuding-pending.json"))
	if err != nil {
		return false
	}
	var pending pendingLocalSnapshot
	if decodeStrict(raw, &pending) != nil || pending.Base != head || projectKey(pending.Project) != projectKey(project) {
		return false
	}
	index, err := localIndexHash(project)
	return err == nil && pending.Index == index
}

func checkpointLocalProject(ctx context.Context, project string) error {
	if info, err := os.Lstat(filepath.Join(project, ".git")); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("local_snapshot_worktree: only the managed original project can be stored automatically")
	}
	headRaw, headErr := localGit(ctx, project, nil, "rev-parse", "--verify", "HEAD")
	head := ""
	if headErr == nil {
		head = strings.TrimSpace(string(headRaw))
	}
	staged, err := localGit(ctx, project, nil, "diff", "--cached", "--name-only", "-z", "--")
	if err != nil {
		return err
	}
	if len(staged) > 0 && !ownsPendingIndex(project, head) {
		return errors.New("local_snapshot_staged: manually staged work has been preserved")
	}
	if err = inspectLocalProjectTree(ctx, project); err != nil {
		return err
	}
	if err = installLocalExcludes(project); err != nil {
		return err
	}
	paths, err := localSourcePaths(ctx, project, headErr == nil)
	if err != nil {
		return err
	}
	if err = commitLocalRecoveryPoint(ctx, project, head, paths); err != nil {
		return err
	}
	if err = os.Remove(filepath.Join(project, ".git", "xiaobuding-pending.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return requireClean(ctx, project, "")
}

func commitLocalRecoveryPoint(ctx context.Context, project, head string, paths []string) error {
	if len(paths) > 0 {
		input := []byte(strings.Join(paths, "\x00") + "\x00")
		if _, err := localGit(ctx, project, input, "add", "--pathspec-from-file=-", "--pathspec-file-nul"); err != nil {
			return err
		}
	}
	if len(paths) > 0 || head == "" {
		if err := savePendingLocalIndex(project, head); err != nil {
			return err
		}
		if _, err := localGit(ctx, project, nil, "-c", "user.name=Xiaobuding Local", "-c", "user.email=local@xiaobuding.invalid", "commit", "--allow-empty", "-m", "chore: save local project recovery point"); err != nil {
			return err
		}
	}
	return nil
}

func savePendingLocalIndex(project, base string) error {
	index, err := localIndexHash(project)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(pendingLocalSnapshot{Project: project, Base: base, Index: index})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(project, ".git", "xiaobuding-pending.json"), raw, 0600)
}

func (e *Engine) prepareLocalRecoveryPoint(ctx context.Context, t *Task) (*localRecoveryPoint, error) {
	if !t.AutoSnapshot {
		return nil, nil
	}
	if err := e.setState(t.ID, statePreparing, ""); err != nil {
		return nil, err
	}
	if err := e.recoverAcceptedParent(ctx, t); err != nil {
		return nil, err
	}
	if err := e.recordLocalInitIntent(ctx, t); err != nil {
		return nil, err
	}
	ownedInit, err := e.ownsLocalInitialization(t)
	if err != nil {
		return nil, err
	}
	point, err := prepareLocalProject(ctx, t.Project, e.Root, ownedInit)
	if err != nil {
		return nil, err
	}
	name := "project-preparation-" + randomID() + ".json"
	if err = e.saveJSON(t.ID, name, point); err != nil {
		return nil, err
	}
	saved, err := e.Store.Update(t.ID, func(saved *Task) error {
		if saved.State == stateCancelled || saved.State == stateCancelling {
			return context.Canceled
		}
		saved.SetupEvidence = append(saved.SetupEvidence, name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	t.SetupEvidence = saved.SetupEvidence
	return point, nil
}
