package reliable

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalSnapshotsPreserveExternalAndManualStaging(t *testing.T) {
	for _, managed := range []bool{false, true} {
		t.Run(map[bool]string{false: "external", true: "managed"}[managed], func(t *testing.T) {
			project := fixtureRepo(t)
			if managed {
				gitTest(t, project, "config", "--local", managedRootKey, project)
			}
			if err := os.WriteFile(filepath.Join(project, "app.txt"), []byte("user staging\n"), 0600); err != nil {
				t.Fatal(err)
			}
			gitTest(t, project, "add", "app.txt")
			index := filepath.Join(project, ".git", "index")
			before, err := os.ReadFile(index)
			if err != nil {
				t.Fatal(err)
			}
			head := gitTest(t, project, "rev-parse", "HEAD")
			_, err = prepareLocalProject(context.Background(), project, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), map[bool]string{false: "external_dirty", true: "staged"}[managed]) {
				t.Fatalf("staged project was not preserved: %v", err)
			}
			after, readErr := os.ReadFile(index)
			if readErr != nil || string(before) != string(after) || head != gitTest(t, project, "rev-parse", "HEAD") {
				t.Fatal("manual staging or HEAD changed")
			}
		})
	}
}

func TestLocalSnapshotRejectsPrivateIgnoreOverrideAndNestedGit(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "private override", true: "nested repository"}[nested], func(t *testing.T) {
			project := plainAppFolder(t)
			if nested {
				if err := os.MkdirAll(filepath.Join(project, "child", ".git"), 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				for name, content := range map[string]string{".gitignore": "!.env\n", ".env": "PRIVATE_FIXTURE=1\n"} {
					if err := os.WriteFile(filepath.Join(project, name), []byte(content), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			_, err := prepareLocalProject(context.Background(), project, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), map[bool]string{false: "private", true: "nested_git"}[nested]) {
				t.Fatalf("unsafe project was admitted: %v", err)
			}
			if !nested {
				if _, err := localGit(context.Background(), project, nil, "rev-parse", "--verify", "HEAD"); err == nil {
					t.Fatal("private content was committed despite ignore conflict")
				}
			}
		})
	}
}

func TestLocalSnapshotDoesNotRebaseWorkingCopiesOrNestedSelections(t *testing.T) {
	project := fixtureRepo(t)
	child := filepath.Join(project, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareLocalProject(context.Background(), child, t.TempDir()); err == nil || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("nested root was initialized: %v", err)
	}
	if _, err := os.Stat(filepath.Join(child, ".git")); !os.IsNotExist(err) {
		t.Fatal("nested selection changed Git metadata")
	}
	root := t.TempDir()
	worktree, _, err := PrepareWorktree(context.Background(), root, project, "candidate")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepareLocalProject(context.Background(), worktree, root); err == nil || !strings.Contains(err.Error(), "boundary") {
		t.Fatalf("internal working copy became a new original: %v", err)
	}
}

func TestPreparingRecoveryAndCancellationKeepZeroImplementationRounds(t *testing.T) {
	h := localFolderHTTP(t, &fixtureAgent{})
	task := &Task{ID: "preparing-task", Project: t.TempDir(), State: statePreparing, AutoSnapshot: true}
	if err := h.Engine.Store.Create(task); err != nil {
		t.Fatal(err)
	}
	if err := h.Engine.Store.Recover(); err != nil {
		t.Fatal(err)
	}
	got, _ := h.Engine.Store.Get(task.ID)
	if got.State != stateInterrupted || got.Round != 0 {
		t.Fatalf("preparation recovery changed budget: %+v", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := prepareLocalProject(ctx, task.Project, h.Engine.Root); err == nil {
		t.Fatal("cancelled preparation continued")
	}
	if _, err := os.Stat(filepath.Join(task.Project, ".git")); !os.IsNotExist(err) {
		t.Fatal("cancelled preparation initialized a repository")
	}
}

func TestInterruptedOwnedInitializationResumesWithoutAdoptingExternalGit(t *testing.T) {
	h := localFolderHTTP(t, &fixtureAgent{})
	project := plainAppFolder(t)
	task := &Task{ID: "recover-local-init", Project: project, Goal: "Update app", Commands: []string{passingCheck}, State: statePreparing, AutoSnapshot: true}
	if err := h.Engine.Store.Create(task); err != nil {
		t.Fatal(err)
	}
	if err := h.Engine.recordLocalInitIntent(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	gitTest(t, project, "init", "--initial-branch=main")
	if err := h.Engine.Store.Recover(); err != nil {
		t.Fatal(err)
	}
	if err := h.Engine.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := h.Engine.Store.Get(task.ID)
	if got.State != stateAwaitingAcceptance || got.Round != 1 {
		t.Fatalf("owned initialization did not recover: %s %s", got.State, got.Reason)
	}
	external := plainAppFolder(t)
	gitTest(t, external, "init", "--initial-branch=main")
	if _, err := prepareLocalProject(context.Background(), external, h.Engine.Root); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("external unfinished Git was adopted: %v", err)
	}
}
