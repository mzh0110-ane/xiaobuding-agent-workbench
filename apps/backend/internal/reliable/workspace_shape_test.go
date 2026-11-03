package reliable

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// @covers AC-RELIABLE-RENEWAL-001.2/.3
func TestSnapshotHandlesFileDirectoryTransitions(t *testing.T) {
	t.Run("tracked file becomes a directory with new source", func(t *testing.T) {
		dir := fixtureRepo(t)
		before, err := TakeSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatal(err)
		}
		oldFile := filepath.Join(dir, "app.txt")
		if err = os.Remove(oldFile); err != nil {
			t.Fatal(err)
		}
		if err = os.Mkdir(oldFile, 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(oldFile, "entry.txt"), []byte("retained refactor\n"), 0600); err != nil {
			t.Fatal(err)
		}
		after, err := TakeSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatalf("a tracked file replaced by a source directory must be representable: %v", err)
		}
		if after.Files["app.txt"] != deletedFingerprint {
			t.Fatalf("the former file must be recorded as deleted: %#v", after.Files)
		}
		if value := after.Files["app.txt/entry.txt"]; value == "" || value == deletedFingerprint {
			t.Fatalf("the new nested source must be fingerprinted: %#v", after.Files)
		}
		if after.Hash == before.Hash || after.Head != before.Head {
			t.Fatal("source shape must change the snapshot without changing the saved Git base")
		}
	})
	t.Run("tracked source directory becomes a file", func(t *testing.T) {
		dir := fixtureRepo(t)
		oldDir := filepath.Join(dir, "module")
		if err := os.Mkdir(oldDir, 0700); err != nil {
			t.Fatal(err)
		}
		oldFile := filepath.Join(oldDir, "entry.txt")
		if err := os.WriteFile(oldFile, []byte("old module\n"), 0600); err != nil {
			t.Fatal(err)
		}
		gitTest(t, dir, "add", ".")
		gitTest(t, dir, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "source directory")
		before, err := TakeSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.Remove(oldFile); err != nil {
			t.Fatal(err)
		}
		if err = os.Remove(oldDir); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(oldDir, []byte("replacement module\n"), 0600); err != nil {
			t.Fatal(err)
		}
		after, err := TakeSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatalf("a tracked directory replaced by a source file must be representable: %v", err)
		}
		if after.Files["module/entry.txt"] != deletedFingerprint {
			t.Fatalf("the former nested file must be recorded as deleted: %#v", after.Files)
		}
		if value := after.Files["module"]; value == "" || value == deletedFingerprint {
			t.Fatalf("the replacement source file must be fingerprinted: %#v", after.Files)
		}
		if after.Hash == before.Hash || after.Head != before.Head {
			t.Fatal("source shape must change the snapshot without changing the saved Git base")
		}
	})
}
