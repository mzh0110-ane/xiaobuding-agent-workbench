package reliable

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	codexOldFixture = "1111111111111111"
	codexNewFixture = "2222222222222222"
)

func codexManagedFixtureRoot(t *testing.T) string {
	t.Helper()
	local, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCALAPPDATA", local)
	root := filepath.Join(local, "OpenAI", "Codex", "bin")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func codexPEFixture(t *testing.T, root, hash string, modified time.Time) string {
	t.Helper()
	path := filepath.Join(root, hash, "codex.exe")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 1024)
	copy(raw, "MZ")
	binary.LittleEndian.PutUint32(raw[60:64], 64)
	copy(raw[64:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(raw[68:70], 0x8664)
	binary.LittleEndian.PutUint16(raw[70:72], 1)
	binary.LittleEndian.PutUint16(raw[84:86], 240)
	binary.LittleEndian.PutUint16(raw[86:88], 0x22)
	binary.LittleEndian.PutUint16(raw[88:90], 0x20b)
	copy(raw[328:336], ".text")
	binary.LittleEndian.PutUint32(raw[344:348], 512)
	binary.LittleEndian.PutUint32(raw[348:352], 512)
	if err := os.WriteFile(path, raw, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCodexDiscoverySelectsNewestValidManagedExecutable(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	oldTime := time.Unix(1_700_000_000, 0)
	old := codexPEFixture(t, root, codexOldFixture, oldTime)
	newest := codexPEFixture(t, root, codexNewFixture, oldTime.Add(time.Hour))
	for _, configured := range []string{old, filepath.Join(root, "3333333333333333", "codex.exe")} {
		path, err := resolveCodexExecutable(configured)
		if err != nil || path != newest {
			t.Fatalf("managed update was not selected: %q %v", path, err)
		}
	}
	if err := os.Remove(old); err != nil {
		t.Fatal(err)
	}
	path, err := resolveCodexExecutable(old)
	if err != nil || path != newest {
		t.Fatalf("removed configured build prevented discovery: %q %v", path, err)
	}
}

func TestCodexDiscoverySkipsIncompleteAndInvalidCandidates(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	valid := codexPEFixture(t, root, codexOldFixture, time.Unix(1_700_000_000, 0))
	fixtures := map[string][]byte{
		"2222222222222222": {},
		"3333333333333333": []byte("a wrapper is not a native executable"),
		"4444444444444444": []byte("MZ"),
	}
	for hash, raw := range fixtures {
		path := filepath.Join(root, hash, "codex.exe")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "5555555555555555"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "6666666666666666", "codex.exe"), 0700); err != nil {
		t.Fatal(err)
	}
	_ = codexPEFixture(t, root, "not-a-release-hash", time.Now())
	path, err := resolveCodexExecutable(valid)
	if err != nil || path != valid {
		t.Fatalf("invalid/incomplete update displaced the valid build: %q %v", path, err)
	}
}

func TestCodexDiscoveryUsesDeterministicPathForEqualModificationTimes(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	modified := time.Unix(1_700_000_000, 0)
	first := codexPEFixture(t, root, codexOldFixture, modified)
	_ = codexPEFixture(t, root, codexNewFixture, modified)
	path, err := resolveCodexExecutable(first)
	if err != nil || path != first {
		t.Fatalf("equal-time candidates were not ordered by path: %q %v", path, err)
	}
}

func TestCodexDiscoveryPreservesExplicitCustomExecutable(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	_ = codexPEFixture(t, root, codexNewFixture, time.Now())
	custom, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path, err := resolveCodexExecutable(custom)
	if err != nil || path != custom {
		t.Fatalf("custom executable was replaced: %q %v", path, err)
	}
	missing := filepath.Join(t.TempDir(), "custom.exe")
	if _, err := resolveCodexExecutable(missing); err == nil {
		t.Fatal("missing custom executable silently fell back to a managed build")
	}
}

func TestCodexDiscoveryDoesNotSearchPath(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	pathDir := t.TempDir()
	_ = codexPEFixture(t, pathDir, codexNewFixture, time.Now())
	t.Setenv("PATH", filepath.Join(pathDir, codexNewFixture))
	if _, err := resolveCodexExecutable(filepath.Join(root, codexOldFixture, "codex.exe")); err == nil {
		t.Fatal("managed discovery accepted a PATH executable")
	}
}

func TestCodexFromHomeRecoversRemovedManagedBuildWithoutWritingSettings(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	selected := codexPEFixture(t, root, codexNewFixture, time.Now())
	home := t.TempDir()
	settings := Settings{DataRoot: filepath.Join(home, "state"), CodexHome: home,
		CodexPath: filepath.Join(root, codexOldFixture, "codex.exe"), Model: "unchanged-model"}
	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(home, "reliable-settings.json")
	if err = os.WriteFile(settingsPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	h, err := FromHome(home)
	if err != nil || h == nil {
		t.Fatalf("stale managed settings blocked startup: %v", err)
	}
	t.Cleanup(func() { _ = h.Engine.Close() })
	if path, err := resolveCodexExecutable(settings.CodexPath); err != nil || path != selected {
		t.Fatalf("configured managed build did not recover: %q %v", path, err)
	}
	after, err := os.ReadFile(settingsPath)
	if err != nil || string(after) != string(raw) || h.Model != settings.Model {
		t.Fatal("discovery rewrote persisted settings or the selected model")
	}
}
