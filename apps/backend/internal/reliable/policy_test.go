package reliable

import (
	"encoding/json"
	"testing"
)

func TestDecisionRejectsUnprovenCandidates(t *testing.T) {
	valid := `{"verdict":"pass","summary":"Checked the actual change","read_files":["app.py"],"findings":[]}`
	for _, tc := range []struct {
		name, raw     string
		checks, fresh bool
		want          Decision
	}{
		{"valid", valid, true, true, Pass},
		{"failed check", valid, false, true, Rework},
		{"stale code", valid, true, false, Block},
		{"malformed", `{`, true, true, Block},
		{"missing verdict", `{"findings":[]}`, true, true, Block},
		{"missing read", `{"verdict":"pass","summary":"ok","read_files":[],"findings":[]}`, true, true, Block},
		{"serious finding", `{"verdict":"pass","summary":"ok","read_files":["app.py"],"findings":[{"severity":"P1","file":"app.py","reason":"wrong result"}]}`, true, true, Rework},
		{"unknown field", valid[:len(valid)-1] + `,"accepted":true}`, true, true, Block},
		{"trailing payload", valid + ` {}`, true, true, Block},
		{"unknown severity", `{"verdict":"pass","summary":"ok","read_files":["app.py"],"findings":[{"severity":"low","file":"app.py","reason":"wrong"}]}`, true, true, Block},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := Decide(json.RawMessage(tc.raw), tc.checks, tc.fresh, []string{"app.py"})
			if got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}
