package reliable

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func indexSnapshotPaths(t *testing.T, dir string, start, end int) {
	t.Helper()
	blob := strings.TrimSpace(gitTest(t, dir, "rev-parse", "HEAD:app.txt"))
	var entries strings.Builder
	for i := start; i < end; i++ {
		_, _ = fmt.Fprintf(&entries, "100644 %s\tvirtual/%05d.txt\n", blob, i)
	}
	cmd := exec.Command("git", "-C", dir, "update-index", "--index-info")
	cmd.Stdin = strings.NewReader(entries.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("index source paths: %v: %s", err, out)
	}
}

func TestSnapshotAdmitsLargeProjectAndRejectsExcessFiles(t *testing.T) {
	dir := fixtureRepo(t)
	indexSnapshotPaths(t, dir, 0, 24000)
	snapshot, err := TakeSnapshot(context.Background(), dir)
	if err != nil {
		t.Fatalf("a 24001-file project must fit the bounded source snapshot: %v", err)
	}
	if len(snapshot.Files) != 24001 {
		t.Fatalf("source entries were omitted: %d", len(snapshot.Files))
	}
	indexSnapshotPaths(t, dir, 24000, 30000)
	if _, err = TakeSnapshot(context.Background(), dir); err == nil || !strings.Contains(err.Error(), "files") {
		t.Fatalf("a 30001-file project must be rejected: %v", err)
	}
}

func TestSnapshotAdmitsLargeTotalAndKeepsByteCeilings(t *testing.T) {
	dir := t.TempDir()
	data := []byte("current source\n")
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, total := range []int64{128 << 20, (256 << 20) - int64(len(data))} {
		if _, err := fileFingerprint(dir, "app.txt", &total); err != nil {
			t.Fatalf("source up to 256 MiB must fit: %v", err)
		}
	}
	total := int64(256<<20) - int64(len(data)) + 1
	if _, err := fileFingerprint(dir, "app.txt", &total); err == nil {
		t.Fatal("source exceeding 256 MiB was accepted")
	}
	file, err := os.Create(filepath.Join(dir, "oversized.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate((16 << 20) + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	total = 0
	if _, err = fileFingerprint(dir, "oversized.bin", &total); err == nil {
		t.Fatal("the 16 MiB single-file ceiling was removed")
	}
}

func TestExistingProjectSnapshotAdmission(t *testing.T) {
	project := os.Getenv("XIAOBUDING_SNAPSHOT_PROJECT")
	if project == "" {
		t.Skip("explicit existing-project snapshot verification is disabled")
	}
	snapshot, err := TakeSnapshot(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("verified %d source entries; snapshot %s", len(snapshot.Files), snapshot.Hash)
}
