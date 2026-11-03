package reliable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const managedRootKey = "xiaobuding.managed-root"

type localRecoveryPoint struct {
	Project string `json:"project"`
	Head    string `json:"head"`
	Managed bool   `json:"managed"`
}

func localProjectBoundary(project, dataRoot string) error {
	info, err := os.Stat(project)
	if err != nil || !info.IsDir() {
		return errors.New("local_snapshot_directory: choose an existing project folder")
	}
	if filepath.Dir(project) == project {
		return errors.New("local_snapshot_boundary: choose one project, not an entire drive")
	}
	if home, homeErr := os.UserHomeDir(); homeErr == nil && projectKey(home) == projectKey(project) {
		return errors.New("local_snapshot_boundary: choose one project, not the user home folder")
	}
	if pathInside(project, dataRoot) || (pathInside(dataRoot, project) && !pathInside(filepath.Join(dataRoot, "projects"), project)) {
		return errors.New("local_snapshot_boundary: choose the original project, not application data or a working copy")
	}
	for _, name := range []string{".codex", "codex-home", "node_modules", ".git", ".ssh"} {
		if strings.EqualFold(filepath.Base(project), name) {
			return errors.New("local_snapshot_boundary: choose project source, not a tool or account folder")
		}
	}
	if _, err = os.Stat(filepath.Join(project, "codex-home", "config.toml")); err == nil {
		return errors.New("local_snapshot_boundary: this is a tools collection; choose a specific project inside it")
	}
	return nil
}

func pathInside(parent, child string) bool {
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func localGit(ctx context.Context, project string, input []byte, args ...string) ([]byte, error) {
	return localGitEnvironment(ctx, project, input, nil, args...)
}

// localGitEnvironment only receives controller-owned environment overrides.
func localGitEnvironment(ctx context.Context, project string, input []byte, extra []string, args ...string) ([]byte, error) {
	prefix := []string{"--literal-pathspecs", "-c", "core.hooksPath=", "-c", "core.fsmonitor=false", "-c", "commit.gpgsign=false", "-C", project}
	cmd := exec.CommandContext(ctx, "git", append(prefix, args...)...)
	cmd.Env = append(isolatedGitEnvironment(), extra...)
	if input != nil {
		cmd.Stdin = strings.NewReader(string(input))
	}
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = gitActivityWriter(ctx, &stderr)
	cmd.Stdout = &stdout
	emitActivity(ctx, "command", "本地存档 · "+args[0], strings.Join(args, " "))
	err := runOwnedCommand(ctx, cmd)
	out := stdout.Bytes()
	if err != nil {
		return out, fmt.Errorf("local Git %s: %w: %.1000s", args[0], err, stderr.Bytes())
	}
	return out, nil
}

func isolatedGitEnvironment() []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(entry), "GIT_") {
			env = append(env, entry)
		}
	}
	return append(env, "GIT_TERMINAL_PROMPT=0")
}

func prepareLocalProject(ctx context.Context, project, dataRoot string, recoverInit ...bool) (*localRecoveryPoint, error) {
	if err := localProjectBoundary(project, dataRoot); err != nil {
		return nil, err
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil, errors.New("local_snapshot_git: Git is unavailable in the local runtime")
	}
	if err := ensureLocalRepository(ctx, project); err != nil {
		return nil, err
	}
	isManaged, err := ensureLocalRecovery(ctx, project, len(recoverInit) > 0 && recoverInit[0])
	if err != nil {
		return nil, err
	}
	head, err := localGit(ctx, project, nil, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	return &localRecoveryPoint{Project: project, Head: strings.TrimSpace(string(head)), Managed: isManaged}, nil
}

func ensureLocalRepository(ctx context.Context, project string) error {
	top, topErr := localGit(ctx, project, nil, "rev-parse", "--show-toplevel")
	if topErr == nil && projectKey(strings.TrimSpace(string(top))) != projectKey(project) {
		return errors.New("local_snapshot_parent: choose the existing repository's project root")
	}
	if topErr == nil {
		return nil
	}
	if _, err := os.Lstat(filepath.Join(project, ".git")); !os.IsNotExist(err) {
		return errors.New("local_snapshot_git_invalid: existing Git metadata could not be read")
	}
	if bare, _ := localGit(ctx, project, nil, "rev-parse", "--is-bare-repository"); strings.TrimSpace(string(bare)) == "true" {
		return errors.New("local_snapshot_boundary: a bare Git repository is not a project folder")
	}
	if err := inspectLocalProjectTree(ctx, project); err != nil {
		return err
	}
	if _, err := localGit(ctx, project, nil, "init", "--initial-branch=main"); err != nil {
		return fmt.Errorf("local_snapshot_init: %w", err)
	}
	_, err := localGit(ctx, project, nil, "config", "--local", managedRootKey, project)
	return err
}

func ensureLocalRecovery(ctx context.Context, project string, recoverInit bool) (bool, error) {
	managed, _ := localGit(ctx, project, nil, "config", "--local", "--get", managedRootKey)
	if len(managed) == 0 && recoverInit {
		if err := recoverLocalInitialization(ctx, project); err != nil {
			return false, err
		}
		managed = []byte(project)
	}
	isManaged := len(managed) > 0 && projectKey(strings.TrimSpace(string(managed))) == projectKey(project)
	if isManaged {
		return true, checkpointLocalProject(ctx, project)
	}
	if err := requireClean(ctx, project, ""); err != nil {
		if _, headErr := localGit(ctx, project, nil, "rev-parse", "--verify", "HEAD"); headErr != nil {
			return false, errors.New("local_snapshot_incomplete: an existing repository has no completed recovery point")
		}
		return false, errors.New("local_snapshot_external_dirty: an existing Git project has unsaved or staged work; its Git history was preserved")
	}
	return false, nil
}
