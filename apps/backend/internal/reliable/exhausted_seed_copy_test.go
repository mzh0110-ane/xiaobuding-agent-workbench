package reliable

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// @covers AC-RELIABLE-RENEWAL-001.2
// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedSeedCopySourceShapes(t *testing.T) {
	for _, directoryToFile := range []bool{true, false} {
		name := "file-to-directory"
		if directoryToFile {
			name = "directory-to-file"
		}
		t.Run(name, func(t *testing.T) {
			e, parent, child := seedCopyFixture(t, directoryToFile)
			oldPath, newPath := "shape", "shape/nested.txt"
			if directoryToFile {
				oldPath, newPath = newPath, oldPath
			}
			if err := os.Remove(filepath.Join(parent.Worktree, filepath.FromSlash(oldPath))); err != nil {
				t.Fatal(err)
			}
			if directoryToFile {
				if err := os.Remove(filepath.Join(parent.Worktree, "shape")); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Mkdir(filepath.Join(parent.Worktree, "shape"), 0700); err != nil {
				t.Fatal(err)
			}
			mustRecoveryWrite(t, filepath.Join(parent.Worktree, filepath.FromSlash(newPath)), "retained shape\n")
			desired := seedCopyDesired(child, map[string]string{oldPath: deletedFingerprint, newPath: digest([]byte("retained shape\n"))})
			indexBefore := gitTest(t, parent.Worktree, "ls-files", "--stage")
			if err := e.copyExhaustedSource(context.Background(), parent, child, desired); err != nil {
				t.Fatalf("valid %s source transition failed: %v", name, err)
			}
			if raw, err := os.ReadFile(filepath.Join(child.Worktree, filepath.FromSlash(newPath))); err != nil || string(raw) != "retained shape\n" {
				t.Fatalf("retained shape missing: %s %v", raw, err)
			}
			if indexAfter := gitTest(t, parent.Worktree, "ls-files", "--stage"); indexAfter != indexBefore {
				t.Fatal("seed copy staged the retained parent")
			}
		})
	}
}

// @covers AC-RELIABLE-RENEWAL-001.3
// @covers AC-RELIABLE-RENEWAL-001.4
func TestExhaustedSeedCopyOutsideStagingDoesNotPolluteRetry(t *testing.T) {
	e, parent, child := seedCopyFixture(t, false)
	controller, staging, err := e.seedCopyController(child)
	if err != nil {
		t.Fatal(err)
	}
	if staging != filepath.Join("evidence", child.ID, "seed-staging") {
		t.Fatalf("controller staging entered candidate source: %s", staging)
	}
	if err = controller.Close(); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(e.Root, staging)
	mustRecoveryWrite(t, filepath.Join(stage, "interrupted-write"), "partial staging bytes")
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "new.txt"), "new untracked source\n")
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "empty.txt"), "")
	desired := seedCopyDesired(child, map[string]string{"new.txt": digest([]byte("new untracked source\n")), "empty.txt": digest(nil)})
	if err := e.copyExhaustedSource(context.Background(), parent, child, desired); err != nil {
		t.Fatal(err)
	}
	if err := e.copyExhaustedSource(context.Background(), parent, child, desired); err != nil {
		t.Fatalf("idempotent retry rejected completed transfer: %v", err)
	}
	if raw, err := os.ReadFile(filepath.Join(child.Worktree, "new.txt")); err != nil || string(raw) != "new untracked source\n" {
		t.Fatalf("new source lost: %s %v", raw, err)
	}
	if matches, err := filepath.Glob(filepath.Join(child.Worktree, ".seed-source-*")); err != nil || len(matches) != 0 {
		t.Fatalf("controller staging polluted source: %v %v", matches, err)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedSeedCopyKeepsIgnoredDirectoryContents(t *testing.T) {
	e, parent, child := seedCopyFixture(t, true)
	common := strings.TrimSpace(gitTest(t, child.Worktree, "rev-parse", "--path-format=absolute", "--git-common-dir"))
	mustRecoveryWrite(t, filepath.Join(common, "info", "exclude"), "shape/ignored.txt\n")
	mustRecoveryWrite(t, filepath.Join(child.Worktree, "shape", "ignored.txt"), "keep ignored content")
	if err := os.Remove(filepath.Join(parent.Worktree, "shape", "nested.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(parent.Worktree, "shape")); err != nil {
		t.Fatal(err)
	}
	mustRecoveryWrite(t, filepath.Join(parent.Worktree, "shape"), "new regular file\n")
	desired := seedCopyDesired(child, map[string]string{"shape/nested.txt": deletedFingerprint, "shape": digest([]byte("new regular file\n"))})
	if err := e.copyExhaustedSource(context.Background(), parent, child, desired); err == nil {
		t.Fatal("ignored contents were silently removed to replace a directory")
	}
	if raw, err := os.ReadFile(filepath.Join(child.Worktree, "shape", "ignored.txt")); err != nil || string(raw) != "keep ignored content" {
		t.Fatalf("ignored content was lost: %s %v", raw, err)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedSeedCopyRejectsSourceSymlink(t *testing.T) {
	e, parent, child := seedCopyFixture(t, false)
	outside := filepath.Join(t.TempDir(), "private.txt")
	mustRecoveryWrite(t, outside, "outside source")
	if err := os.Symlink(outside, filepath.Join(parent.Worktree, "link.txt")); err != nil {
		t.Skipf("source symlink fixture requires host privilege: %v", err)
	}
	desired := seedCopyDesired(child, map[string]string{"link.txt": digest([]byte("outside source"))})
	if err := e.copyExhaustedSource(context.Background(), parent, child, desired); err == nil {
		t.Fatal("source symlink was followed into a different directory")
	}
	if _, err := os.Lstat(filepath.Join(child.Worktree, "link.txt")); !os.IsNotExist(err) {
		t.Fatalf("unsafe source produced a child file: %v", err)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.3
func TestExhaustedSeedCopyPreservesUnexpectedFiles(t *testing.T) {
	e, parent, child := seedCopyFixture(t, true)
	mustRecoveryWrite(t, filepath.Join(child.Worktree, "shape", "outside.txt"), "keep my edits")
	desired := seedCopyDesired(child, nil)
	if err := e.copyExhaustedSource(context.Background(), parent, child, desired); err == nil {
		t.Fatal("unrelated candidate edit was silently adopted")
	}
	if raw, err := os.ReadFile(filepath.Join(child.Worktree, "shape", "outside.txt")); err != nil || string(raw) != "keep my edits" {
		t.Fatalf("unrelated candidate edit was lost: %s %v", raw, err)
	}
}

func seedCopyFixture(t *testing.T, directory bool) (*Engine, *Task, *Task) {
	t.Helper()
	project := fixtureRepo(t)
	name := "shape"
	if directory {
		if err := os.Mkdir(filepath.Join(project, name), 0700); err != nil {
			t.Fatal(err)
		}
		name += "/nested.txt"
	}
	mustRecoveryWrite(t, filepath.Join(project, filepath.FromSlash(name)), "original shape\n")
	gitTest(t, project, "add", ".")
	gitTest(t, project, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "shape baseline")
	e := &Engine{Root: t.TempDir()}
	parentDir, base, err := PrepareWorktree(context.Background(), e.Root, project, "copy-parent")
	if err != nil {
		t.Fatal(err)
	}
	childDir, _, err := PrepareWorktree(context.Background(), e.Root, project, "copy-child")
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := TakeSnapshot(context.Background(), childDir)
	if err != nil {
		t.Fatal(err)
	}
	return e, &Task{ID: "copy-parent", Project: project, Worktree: parentDir, Base: base},
		&Task{ID: "copy-child", Project: project, Worktree: childDir, Base: base, Baseline: baseline}
}

func seedCopyDesired(child *Task, delta map[string]string) *Snapshot {
	desired := &Snapshot{Head: child.Baseline.Head, Files: make(map[string]string)}
	for name, fingerprint := range child.Baseline.Files {
		desired.Files[name] = fingerprint
	}
	for name, fingerprint := range delta {
		desired.Files[name] = fingerprint
	}
	return desired
}
