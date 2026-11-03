package reliable

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// @covers AC-RELIABLE-ATTACHMENTS-001.3
func TestAttachmentStorageRejectsRedirectedDirectory(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	ref := storedAttachment(t, e, "source.bin", []byte("kept bytes"))
	dir := filepath.Join(e.Root, "attachments", ref.ID)
	external := filepath.Join(t.TempDir(), "outside")
	if err := os.Rename(dir, external); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("cmd.exe", "/c", "mklink", "/J", dir, external).CombinedOutput()
	if err != nil {
		t.Fatalf("create fixture junction: %v: %s", err, out)
	}
	t.Cleanup(func() {
		if err := os.Remove(dir); err != nil {
			t.Error(err)
		}
	})
	if _, err = e.resolveAttachments([]string{ref.ID}); err == nil {
		t.Fatal("redirected attachment was trusted")
	}
	data, err := os.ReadFile(filepath.Join(external, "content"))
	if err != nil || string(data) != "kept bytes" {
		t.Fatalf("external bytes were modified: %s %v", data, err)
	}
}
