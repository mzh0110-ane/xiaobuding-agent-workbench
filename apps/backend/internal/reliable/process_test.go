package reliable

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAgentSessionRequiresOneActualThread(t *testing.T) {
	good := []byte("{\"type\":\"thread.started\",\"thread_id\":\"actual-thread\"}\n{\"type\":\"turn.completed\"}\n")
	id, err := parseSession(good)
	if err != nil || id != "actual-thread" {
		t.Fatalf("actual session missing: %s %v", id, err)
	}
	for _, raw := range []string{"", `{"session":"self-reported"}`, string(good) + "{\"type\":\"thread.started\",\"thread_id\":\"other-thread\"}\n", string(good) + "{\"type\":\"error\",\"message\":\"failed\"}\n"} {
		if _, err := parseSession([]byte(raw)); err == nil {
			t.Fatalf("accepted ambiguous/failed event stream: %s", raw)
		}
	}
}

func TestCheckLogsPreserveChinese(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "check.log")
	check, err := runCheck(context.Background(), dir, `python -c "print('\u4e2d\u6587')"`, logPath, nil)
	if err != nil || check.ExitCode != 0 {
		t.Fatalf("check failed: %+v %v", check, err)
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !utf8.Valid(raw) || !strings.Contains(string(raw), "中文") {
		t.Fatalf("Chinese log was corrupted: %q", raw)
	}
}

func TestCheckDiscoveryRejectsPlaceholderTests(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"echo \"Error: no test specified\" && exit 1"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := discoverChecks(dir); len(got) != 0 {
		t.Fatalf("placeholder accepted: %v", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"test":"node --test","build":"vite build"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := discoverChecks(dir); len(got) != 2 {
		t.Fatalf("real checks missing: %v", got)
	}
}
