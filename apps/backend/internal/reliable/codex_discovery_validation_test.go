package reliable

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodexDiscoveryRejectsMalformedNativeHeaders(t *testing.T) {
	changes := map[string]func([]byte){
		"offset":      func(raw []byte) { binary.LittleEndian.PutUint32(raw[60:64], 1<<30) },
		"signature":   func(raw []byte) { raw[64] = 'X' },
		"machine":     func(raw []byte) { binary.LittleEndian.PutUint16(raw[68:70], 0) },
		"no-sections": func(raw []byte) { binary.LittleEndian.PutUint16(raw[70:72], 0) },
		"not-exe":     func(raw []byte) { binary.LittleEndian.PutUint16(raw[86:88], 0) },
		"dll":         func(raw []byte) { binary.LittleEndian.PutUint16(raw[86:88], 0x2002) },
		"optional":    func(raw []byte) { binary.LittleEndian.PutUint16(raw[84:86], 0xffff) },
		"magic":       func(raw []byte) { binary.LittleEndian.PutUint16(raw[88:90], 0) },
		"section-count": func(raw []byte) {
			binary.LittleEndian.PutUint16(raw[70:72], 0xffff)
		},
		"section-data-missing": func(raw []byte) {
			binary.LittleEndian.PutUint32(raw[344:348], 0)
		},
		"section-data-offset": func(raw []byte) {
			binary.LittleEndian.PutUint32(raw[348:352], 1)
		},
		"section-data-size": func(raw []byte) {
			binary.LittleEndian.PutUint32(raw[344:348], 0xffff_ffff)
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			root := codexManagedFixtureRoot(t)
			path := codexPEFixture(t, root, codexNewFixture, time.Now())
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			change(raw)
			if err = os.WriteFile(path, raw, 0700); err != nil {
				t.Fatal(err)
			}
			if _, err = resolveCodexExecutable(path); err == nil {
				t.Fatal("malformed native executable was accepted")
			}
		})
	}
}

func TestCodexDiscoverySkipsNewerTruncatedExecutable(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	modified := time.Unix(1_700_000_000, 0)
	old := codexPEFixture(t, root, codexOldFixture, modified)
	newest := codexPEFixture(t, root, codexNewFixture, modified.Add(time.Hour))
	for _, size := range []int64{350, 512, 1023} {
		if err := os.Truncate(newest, size); err != nil {
			t.Fatal(err)
		}
		if path, err := resolveCodexExecutable(old); err != nil || path != old {
			t.Fatalf("truncated native update displaced the working build: %q %v", path, err)
		}
	}
}

func TestCodexDiscoveryBoundsDirectoryEntries(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	path := codexPEFixture(t, root, codexOldFixture, time.Now())
	for i := range codexDiscoveryLimit {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("ignored-%d", i)), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := resolveCodexExecutable(path); err == nil {
		t.Fatal("managed discovery scanned beyond its directory entry bound")
	}
}

func TestCodexDiscoveryRejectsRedirectedManagedRoot(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	target := t.TempDir()
	_ = codexPEFixture(t, target, codexNewFixture, time.Now())
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, root); err != nil {
		t.Skipf("symbolic link creation unavailable: %v", err)
	}
	if _, err := resolveCodexExecutable(filepath.Join(root, codexOldFixture, "codex.exe")); err == nil {
		t.Fatal("redirected managed root was accepted")
	}
}

func TestCodexFromHomeDefersMissingManagedExecutableButRejectsMissingCustomPath(t *testing.T) {
	root := codexManagedFixtureRoot(t)
	home := t.TempDir()
	settings := Settings{DataRoot: filepath.Join(home, "state"), CodexHome: home,
		CodexPath: filepath.Join(root, codexOldFixture, "codex.exe")}
	settingsPath := filepath.Join(home, "reliable-settings.json")
	write := func() {
		t.Helper()
		raw, err := json.Marshal(settings)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(settingsPath, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	h, err := FromHome(home)
	if err != nil || h == nil {
		t.Fatalf("unused missing managed Codex prevented startup: %v", err)
	}
	if err = h.Engine.Close(); err != nil {
		t.Fatal(err)
	}
	settings.CodexPath = filepath.Join(t.TempDir(), "custom.exe")
	write()
	if _, err = FromHome(home); err == nil {
		t.Fatal("missing explicit custom Codex path silently fell back")
	}
}
