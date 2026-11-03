package reliable

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func harnessEvents(t *testing.T, dir string) []map[string]any {
	t.Helper()
	return []map[string]any{
		{"type": "session", "sessionId": "session-native-unique", "cwd": dir},
		{"type": "status", "phase": "turn_start", "turn": 0},
		{"type": "text", "text": "Inspecting actual source"},
		{"type": "status", "phase": "turn_end", "turn": 0, "reason": map[string]string{"kind": "completed"}},
		{"type": "final", "text": `{"summary":"Native report","read_files":["app.txt"],"docs_impact":"none","residual_risks":[]}`},
	}
}

func streamBytes(t *testing.T, events []map[string]any) []byte {
	t.Helper()
	var lines []string
	for _, event := range events {
		raw, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(raw))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func TestHarnessRequiresItsNativeSessionAndCompletedFinalReport(t *testing.T) {
	dir := t.TempDir()
	result, err := parseHarnessEvents(streamBytes(t, harnessEvents(t, dir)), dir)
	if err != nil || result == nil || result.Session != "session-native-unique" || !json.Valid(result.Output) {
		t.Fatalf("native Harness session was not recognized: %+v %v", result, err)
	}
}

func TestHarnessRejectsAmbiguousFailedOrWrongProjectStreams(t *testing.T) {
	for _, kind := range []string{"no-session", "two-sessions", "wrong-project", "failed-turn", "incomplete-turn", "error", "no-final", "two-finals", "bad-report", "after-final"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			events := harnessEvents(t, dir)
			switch kind {
			case "no-session":
				events = events[1:]
			case "two-sessions":
				events = append([]map[string]any{events[0]}, events...)
			case "wrong-project":
				events[0]["cwd"] = filepath.Join(dir, "another-project")
			case "failed-turn":
				events[3]["reason"] = map[string]string{"kind": "error"}
			case "incomplete-turn":
				events = append(events[:4], map[string]any{"type": "status", "phase": "turn_start"}, events[4])
			case "error":
				events[2] = map[string]any{"type": "error", "message": "provider failed"}
			case "no-final":
				events = events[:4]
			case "two-finals":
				events = append(events, events[4])
			case "bad-report":
				events[4]["text"] = "I promise it works"
			case "after-final":
				events = append(events, map[string]any{"type": "tool_call", "tool": "write"})
			}
			if _, err := parseHarnessEvents(streamBytes(t, events), dir); err == nil {
				t.Fatal("unproven native Harness execution was accepted")
			}
		})
	}
}
