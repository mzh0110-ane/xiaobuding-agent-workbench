package reliable

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (e *Engine) copyExhaustedSource(ctx context.Context, parent, child *Task, desired *Snapshot) error {
	current, err := TakeSnapshot(ctx, child.Worktree)
	if err != nil {
		return err
	}
	if err = validatePartialSeed(child.Baseline, desired, current); err != nil {
		return err
	}
	source, err := os.OpenRoot(parent.Worktree)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	target, err := os.OpenRoot(child.Worktree)
	if err != nil {
		return err
	}
	defer func() { _ = target.Close() }()
	controller, staging, err := e.seedCopyController(child)
	if err != nil {
		return err
	}
	defer func() { _ = controller.Close() }()
	paths := changedFiles(child.Baseline, desired)
	if err = deleteSeedSources(ctx, target, paths, current, desired); err != nil {
		return err
	}
	for _, name := range paths {
		if err = ctx.Err(); err != nil {
			return err
		}
		if normalizedSeedFingerprint(desired.Files[name]) == deletedFingerprint || normalizedSeedFingerprint(current.Files[name]) == normalizedSeedFingerprint(desired.Files[name]) {
			continue
		}
		if err = copySeedFile(source, target, controller, staging, child.ID, name, desired.Files[name]); err != nil {
			return err
		}
	}
	return copySeedResources(ctx, child.Worktree, desired.AvailableResources)
}

func (e *Engine) seedCopyController(child *Task) (*os.Root, string, error) {
	relative, err := filepath.Rel(e.Root, child.Worktree)
	if err != nil || !safeRelative(child.ID) || strings.Contains(child.ID, "/") || relative != filepath.Join("worktrees", child.ID) {
		return nil, "", errors.New("batch_seed_invalid: child source is outside its owned directory")
	}
	root, err := os.OpenRoot(e.Root)
	if err != nil {
		return nil, "", err
	}
	staging := filepath.Join("evidence", child.ID, "seed-staging")
	if err = ensureSeedDirectories(root, filepath.ToSlash(filepath.Join(staging, "reserved"))); err != nil {
		_ = root.Close()
		return nil, "", err
	}
	return root, staging, nil
}

func deleteSeedSources(ctx context.Context, target *os.Root, paths []string, current, desired *Snapshot) error {
	deleted := make([]string, 0)
	for _, name := range paths {
		if normalizedSeedFingerprint(desired.Files[name]) == deletedFingerprint {
			deleted = append(deleted, name)
		}
	}
	sort.Slice(deleted, func(i, j int) bool { return len(deleted[i]) > len(deleted[j]) })
	for _, name := range deleted {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !safeRelative(name) || protectedPath(name) {
			return errors.New("batch_seed_invalid: unsafe inherited source path")
		}
		if normalizedSeedFingerprint(current.Files[name]) == deletedFingerprint {
			continue
		}
		if err := target.Remove(filepath.FromSlash(name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return pruneEmptySeedDirectories(target, deleted)
}

func pruneEmptySeedDirectories(root *os.Root, deleted []string) error {
	for _, dir := range seedDeletionDirectories(deleted) {
		if err := removeEmptySeedDirectory(root, dir); err != nil {
			return err
		}
	}
	return nil
}

func seedDeletionDirectories(deleted []string) []string {
	directories := make(map[string]bool)
	for _, name := range deleted {
		for dir := filepath.Dir(filepath.FromSlash(name)); dir != "."; dir = filepath.Dir(dir) {
			directories[dir] = true
		}
	}
	ordered := make([]string, 0, len(directories))
	for dir := range directories {
		ordered = append(ordered, dir)
	}
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	return ordered
}

func removeEmptySeedDirectory(root *os.Root, dir string) error {
	info, err := root.Lstat(dir)
	if os.IsNotExist(err) || (err == nil && !info.IsDir()) {
		return nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("batch_seed_invalid: inherited source directory is unsafe")
	}
	file, err := root.Open(dir)
	if err != nil {
		return err
	}
	entries, readErr := file.ReadDir(1)
	closeErr := file.Close()
	if closeErr != nil || (readErr != nil && !errors.Is(readErr, io.EOF)) {
		return errors.Join(readErr, closeErr)
	}
	if len(entries) == 0 {
		return root.Remove(dir)
	}
	return nil
}

func normalizedSeedFingerprint(value string) string {
	if value == "" {
		return deletedFingerprint
	}
	return value
}

func validatePartialSeed(baseline, desired, current *Snapshot) error {
	if current.Head != baseline.Head || current.ResourceHash != baseline.ResourceHash {
		return errors.New("batch_seed_invalid: interrupted candidate history or resources changed")
	}
	for _, name := range unionPaths(changedFiles(baseline, current), changedFiles(baseline, desired)) {
		got := normalizedSeedFingerprint(current.Files[name])
		if got != normalizedSeedFingerprint(baseline.Files[name]) && got != normalizedSeedFingerprint(desired.Files[name]) {
			return fmt.Errorf("batch_seed_invalid: preserve unexpected candidate edit before retrying: %s", name)
		}
	}
	return nil
}

func copySeedFile(source, target, controller *os.Root, staging, childID, name, want string) error {
	if !safeRelative(name) || protectedPath(name) {
		return errors.New("batch_seed_invalid: unsafe inherited source path")
	}
	path := filepath.FromSlash(name)
	info, err := source.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16<<20 {
		return errors.New("batch_parent_changed: inherited source is missing or unsafe")
	}
	file, err := source.Open(path)
	if err != nil {
		return err
	}
	data := make([]byte, info.Size())
	_, readErr := file.ReadAt(data, 0)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || digest(data) != want {
		return errors.New("batch_parent_changed: inherited source bytes changed during transfer")
	}
	if err = ensureSeedDirectories(target, name); err != nil {
		return err
	}
	temporary := filepath.Join(staging, ".seed-source-"+randomID())
	out, err := controller.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer func() { _ = controller.Remove(temporary) }()
	_, writeErr := out.Write(data)
	if err = errors.Join(writeErr, out.Sync(), out.Close()); err != nil {
		return err
	}
	return controller.Rename(temporary, filepath.Join("worktrees", childID, path))
}

func ensureSeedDirectories(root *os.Root, name string) error {
	parent := ""
	parts := strings.Split(name, "/")
	for _, part := range parts[:len(parts)-1] {
		parent = filepath.Join(parent, part)
		info, err := root.Lstat(parent)
		if os.IsNotExist(err) {
			if err = root.Mkdir(parent, 0700); err != nil {
				return err
			}
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("batch_seed_invalid: inherited source parent is unsafe")
		}
	}
	return nil
}

func copySeedResources(ctx context.Context, worktree string, available map[string]string) error {
	paths := make([]string, 0, len(available))
	for path := range available {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for start := 0; start < len(paths); start += 64 {
		end := min(start+64, len(paths))
		if err := materializeResources(ctx, worktree, paths[start:end]); err != nil {
			return err
		}
	}
	return nil
}
