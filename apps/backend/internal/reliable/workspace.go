package reliable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const deletedFingerprint = "deleted"

type Snapshot struct {
	ResourceHash       string            `json:"resource_hash,omitempty"`
	AvailableResources map[string]string `json:"available_resources,omitempty"`
	Hash               string            `json:"hash"`
	Head               string            `json:"head"`
	Files              map[string]string `json:"files"`
}

func git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.hooksPath=", "-c", "core.fsmonitor=false", "-C", dir}, args...)...)
	cmd.Env = isolatedGitEnvironment()
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = gitActivityWriter(ctx, &stderr)
	emitActivity(ctx, "command", "本地 Git · "+args[0], strings.Join(args, " "))
	err := runOwnedCommand(ctx, cmd)
	out := stdout.Bytes()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %.1000s", args[0], err, stderr.Bytes())
	}
	return out, nil
}

func safeRelative(path string) bool {
	if path == "" || filepath.IsAbs(path) || strings.ContainsAny(path, "\\:\x00") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." || part == ".git" || part == "" {
			return false
		}
	}
	return true
}

func TakeSnapshot(ctx context.Context, dir string) (*Snapshot, error) {
	if manifest, marker, err := loadResourceProfile(ctx, dir); err != nil || manifest != nil {
		if err != nil {
			return nil, err
		}
		return takeResourceSnapshot(ctx, dir, manifest, marker)
	}
	head, err := git(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	raw, err := git(ctx, dir, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	committed, err := git(ctx, dir, "ls-tree", "-r", "--name-only", "-z", "HEAD")
	if err != nil {
		return nil, err
	}
	raw = append(raw, committed...)
	paths := make(map[string]bool)
	for _, part := range bytes.Split(raw, []byte{0}) {
		if len(part) > 0 {
			paths[string(part)] = true
		}
	}
	if len(paths) > 30000 {
		return nil, errors.New("source snapshot exceeds 30000 files")
	}
	names := make([]string, 0, len(paths))
	for path := range paths {
		names = append(names, path)
	}
	sort.Strings(names)
	snap := &Snapshot{Head: strings.TrimSpace(string(head)), Files: make(map[string]string)}
	h := sha256.New()
	_, _ = fmt.Fprintln(h, snap.Head)
	var total int64
	for _, path := range names {
		if !safeRelative(path) {
			return nil, fmt.Errorf("unsafe source path: %q", path)
		}
		fingerprint, err := fileFingerprint(dir, path, &total)
		if err != nil {
			return nil, err
		}
		snap.Files[path] = fingerprint
		_, _ = fmt.Fprintf(h, "%d:%s:%s\n", len(path), path, fingerprint)
	}
	snap.Hash = hex.EncodeToString(h.Sum(nil))
	return snap, nil
}

func fileFingerprint(dir, path string, total *int64) (string, error) {
	info, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(path)))
	if os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR) {
		return deletedFingerprint, nil
	}
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return deletedFingerprint, nil
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("source links and submodules are unsupported: %s", path)
	}
	*total += info.Size()
	if info.Size() > 16<<20 || *total > 256<<20 {
		return "", errors.New("source snapshot exceeds 256 MiB total or 16 MiB per-file limit")
	}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func requireClean(ctx context.Context, project, base string) error {
	head, err := git(ctx, project, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if base != "" && strings.TrimSpace(string(head)) != base {
		return errors.New("original project HEAD changed")
	}
	status, err := git(ctx, project, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return err
	}
	if len(status) > 0 {
		return errors.New("original project has uncommitted changes; they must be preserved before continuing")
	}
	return nil
}

func PrepareWorktree(ctx context.Context, root, project, id string) (string, string, error) {
	if !safeRelative(id) || strings.Contains(id, "/") {
		return "", "", errors.New("invalid task ID")
	}
	top, err := git(ctx, project, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", err
	}
	if projectKey(strings.TrimSpace(string(top))) != projectKey(project) {
		return "", "", errors.New("choose the Git repository root")
	}
	if err := preflightWorktree(ctx, project); err != nil {
		return "", "", err
	}
	if err := requireClean(ctx, project, ""); err != nil {
		return "", "", err
	}
	head, err := git(ctx, project, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	dir := filepath.Join(root, "worktrees", id)
	if err := os.MkdirAll(filepath.Dir(dir), 0700); err != nil {
		return "", "", err
	}
	base := strings.TrimSpace(string(head))
	manifest, err := resourceInventory(ctx, project)
	if err != nil {
		return "", "", err
	}
	if manifest != nil {
		return prepareResourceWorktree(ctx, root, project, id, dir, base, manifest)
	}
	emitActivity(ctx, "phase", "正在创建独立工作副本", "这一步只在本机复制文件，不会上传 GitHub。")
	if _, statErr := os.Stat(dir); statErr == nil {
		if err = recoverInitialWorktree(ctx, project, dir, id, base); err != nil {
			return "", "", err
		}
		return dir, base, nil
	} else if !os.IsNotExist(statErr) {
		return "", "", statErr
	}
	if _, err := git(ctx, project, "worktree", "add", "-b", "xiaobuding/"+id, dir, base); err != nil {
		return "", "", err
	}
	return dir, base, nil
}

func preflightWorktree(ctx context.Context, project string) error {
	if _, err := os.Stat(filepath.Join(project, ".git")); os.IsNotExist(err) {
		return nil
	}
	emitActivity(ctx, "phase", "检查项目规模", "先核对 Git 已跟踪的文件数量，再准备副本。")
	tracked, err := git(ctx, project, "ls-files", "--cached", "-z")
	if err != nil {
		return err
	}
	if bytes.Count(tracked, []byte{0}) > 30000 {
		_, err = resourceInventory(ctx, project)
		return err
	}
	return nil
}

func recoverInitialWorktree(ctx context.Context, project, dir, id, base string) error {
	common, err := git(ctx, dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	original, err := git(ctx, project, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	branch, err := git(ctx, dir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return err
	}
	if projectKey(strings.TrimSpace(string(common))) != projectKey(strings.TrimSpace(string(original))) || strings.TrimSpace(string(branch)) != "xiaobuding/"+id {
		return errors.New("existing working directory does not belong to this task")
	}
	return requireClean(ctx, dir, base)
}

func ExportPatch(ctx context.Context, dir, base, target string) error {
	args := []string{"add", "-A", "--", "."}
	if m, _, err := loadResourceProfile(ctx, dir); err != nil {
		return err
	} else if m != nil {
		args = []string{"add", "--sparse", "-A", "--", "."}
		if _, err = TakeSnapshot(ctx, dir); err != nil {
			return err
		}
	}
	if _, err := git(ctx, dir, args...); err != nil {
		return err
	}
	if m, _, err := loadResourceProfile(ctx, dir); err != nil {
		return err
	} else if m != nil {
		if err = validateResourceIndex(ctx, dir, m); err != nil {
			return err
		}
	}
	raw, err := git(ctx, dir, "diff", "--cached", "--binary", "--no-ext-diff", base, "--")
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return errors.New("implementation produced no changes")
	}
	return os.WriteFile(target, raw, 0600)
}

func ApplyPatch(ctx context.Context, project, base, patch string) error {
	if err := requireClean(ctx, project, base); err != nil {
		return err
	}
	if _, err := git(ctx, project, "apply", "--check", "--", patch); err != nil {
		return err
	}
	_, err := git(ctx, project, "apply", "--", patch)
	return err
}

func changedFiles(before, after *Snapshot) []string {
	set := make(map[string]bool)
	for p, h := range after.Files {
		if before.Files[p] != h {
			set[p] = true
		}
	}
	for p := range before.Files {
		if _, ok := after.Files[p]; !ok {
			set[p] = true
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func checkScope(changed, allowed []string) error {
	for _, path := range changed {
		if !safeRelative(path) {
			return fmt.Errorf("unsafe changed path: %s", path)
		}
		if protectedPath(path) {
			return fmt.Errorf("protected configuration was changed: %s", path)
		}
		if !withinScope(path, allowed) {
			return fmt.Errorf("change outside audited scope: %s", path)
		}
	}
	return nil
}

func protectedPath(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return name == "agents.md" || name == "claude.md" || name == "auth.json" || name == ".xiaobuding.json" || strings.HasPrefix(name, ".env") || strings.HasPrefix(path, ".codex/") || strings.HasPrefix(path, ".agents/")
}
