package reliable

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=", "-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s %v", args, out, err)
	}
	return string(out)
}

func fixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitTest(t, dir, "init")
	gitTest(t, dir, "config", "core.autocrlf", "false")
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("before\n"), 0600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, dir, "add", ".")
	gitTest(t, dir, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "initial")
	return dir
}

func TestSnapshotDetectsNewFilesAndSourceDrift(t *testing.T) {
	dir := fixtureRepo(t)
	ctx := context.Background()
	a, err := TakeSnapshot(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("untracked"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := TakeSnapshot(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Hash == b.Hash {
		t.Fatal("untracked file was omitted")
	}
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := TakeSnapshot(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if b.Hash == c.Hash {
		t.Fatal("source change was omitted")
	}
}

func TestPatchApplicationRequiresUnchangedCleanOriginal(t *testing.T) {
	project := fixtureRepo(t)
	ctx := context.Background()
	root := t.TempDir()
	work, base, err := PrepareWorktree(ctx, root, project, "candidate")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "app.txt"), []byte("reviewed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(t.TempDir(), "change.patch")
	if err := ExportPatch(ctx, work, base, patch); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "unrelated.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPatch(ctx, project, base, patch); err == nil {
		t.Fatal("dirty original was overwritten")
	}
	raw, _ := os.ReadFile(filepath.Join(project, "app.txt"))
	if string(raw) != "before\n" {
		t.Fatal("failed admission changed original")
	}
	if err := os.Remove(filepath.Join(project, "unrelated.txt")); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPatch(ctx, project, base, patch); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(project, "app.txt"))
	if string(raw) != "reviewed\n" {
		t.Fatalf("patch was not applied: %s", raw)
	}
}
