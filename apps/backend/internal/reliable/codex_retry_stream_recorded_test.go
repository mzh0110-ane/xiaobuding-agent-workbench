package reliable

import (
	"os"
	"testing"
)

func TestParseSessionRecordedRecoveredNativeTurn(t *testing.T) {
	path := os.Getenv("XIAOBUDING_RECORDED_CODEX_STREAM")
	if path == "" {
		t.Skip("explicit recorded native stream fixture is not configured")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := parseSession(raw); err != nil || id == "" {
		t.Fatalf("completed native reconnect sequence was rejected: %v", err)
	}
	t.Log("verified original reconnect and completion events without rerunning or changing the native session")
}
