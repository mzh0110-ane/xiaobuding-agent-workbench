package reliable

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const desktopOpen = "open"

func canArchiveTask(state string) bool {
	switch state {
	case stateAccepted, stateCancelled, stateExhausted, stateBlocked, stateInterrupted, stateAwaitingAcceptance, stateInspected:
		return true
	default:
		return false
	}
}

func (e *Engine) ArchiveTask(id string, archived bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, active := e.active[id]; active {
		return errors.New("task_running: stop or finish the current execution before archiving")
	}
	_, err := e.Store.Update(id, func(t *Task) error {
		if archived {
			if !canArchiveTask(t.State) {
				return errors.New("task_running: this task cannot be archived during execution")
			}
			if t.ArchivedAt == "" {
				t.ArchivedAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
		} else {
			t.ArchivedAt = ""
		}
		return nil
	})
	return err
}

func (h *HTTP) openTaskFolder(ctx context.Context, id, action string) error {
	t, err := h.Engine.Store.Get(id)
	if err != nil {
		return err
	}
	path, err := h.taskFolder(t, action)
	if err != nil {
		return err
	}
	opener := h.OpenFolder
	if opener == nil {
		opener = openDesktopFolder
	}
	return opener(ctx, path)
}

func (h *HTTP) taskFolder(t *Task, action string) (string, error) {
	path, boundary := t.Project, ""
	switch action {
	case desktopOpen:
		path, boundary = t.Worktree, filepath.Join(h.Engine.Root, "worktrees")
	case "open-evidence":
		path, boundary = filepath.Join(h.Engine.Root, "evidence", t.ID), filepath.Join(h.Engine.Root, "evidence")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("folder_unavailable: this task folder has not been created")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.New("folder_unavailable: this task folder no longer exists")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("folder_unavailable: this task path is not a directory")
	}
	if boundary != "" {
		root, rootErr := filepath.EvalSymlinks(boundary)
		if rootErr != nil || !pathInside(projectKey(root), projectKey(resolved)) {
			return "", errors.New("folder_unavailable: the task folder is outside its storage boundary")
		}
	}
	return resolved, nil
}

func openDesktopFolder(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name := "xdg-open"
	switch runtime.GOOS {
	case "windows":
		name = "explorer.exe"
	case "darwin":
		name = desktopOpen
	}
	cmd := exec.Command(name, path)
	if err := cmd.Start(); err != nil {
		return errors.New("folder_open_failed: the desktop file manager could not start")
	}
	return cmd.Process.Release()
}
