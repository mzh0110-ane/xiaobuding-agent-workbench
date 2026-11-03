package reliable

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestProjectHandoffSeparatesEvidenceAndHistoricalClaims(t *testing.T) {
	e, task, h, _ := editableProjectFixture(t)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.RequirementRevision, saved.AuditedRevision = 2, 1
		saved.Base, saved.Snapshot = "old-base", "candidate-hash"
		saved.State = stateBlocked
		saved.Reason = "Current source needs a fresh check"
		saved.Audit = &Audit{Facts: []string{"Source fact from app.txt"}, Inferences: []string{"A possible explanation"}, Unknowns: []string{"Real visual result remains unchecked"}}
		saved.Review = json.RawMessage(`{"verdict":"rework","summary":"Still needs work","read_files":["app.txt"],"findings":[{"severity":"P1","file":"app.txt","reason":"Unresolved requirement"}]}`)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/project", "", "")
	var wire editableProjectWire
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Handoff) == 0 || string(wire.Handoff) == "null" {
		t.Fatal("project has no provenance-bearing handoff")
	}
	var handoff struct {
		TaskID              string   `json:"task_id"`
		RequirementRevision int      `json:"requirement_revision"`
		AuditedRevision     int      `json:"audited_revision"`
		VerifiedBatch       bool     `json:"verified_batch"`
		RequirementsCurrent bool     `json:"requirements_current"`
		Warnings            []string `json:"warnings"`
		OpenIssues          []struct {
			Kind, Text string
			Verified   bool
		} `json:"open_issues"`
	}
	if err = json.Unmarshal(wire.Handoff, &handoff); err != nil {
		t.Fatal(err)
	}
	if handoff.TaskID != task.ID || handoff.RequirementRevision != 2 || handoff.AuditedRevision != 1 || handoff.VerifiedBatch || handoff.RequirementsCurrent || len(handoff.Warnings) == 0 {
		t.Fatalf("stale handoff was misrepresented: %+v", handoff)
	}
	found := false
	for _, issue := range handoff.OpenIssues {
		if strings.Contains(issue.Text, "Unresolved requirement") {
			found = true
			if issue.Verified {
				t.Fatal("model finding became independently verified fact")
			}
		}
	}
	if !found {
		t.Fatal("unresolved finding disappeared from handoff")
	}
}

func TestProjectHandoffRemainsBounded(t *testing.T) {
	e, task, h, _ := editableProjectFixture(t)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.Goal = strings.Repeat("完整需求", 800)
		saved.Audit = &Audit{Facts: []string{strings.Repeat("Source claim ", 1000)}, Unknowns: []string{strings.Repeat("Unknown ", 1000)}}
		saved.Review = json.RawMessage(`{"verdict":"blocked","summary":"Needs evidence","read_files":[],"findings":[]}`)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/project", "", "")
	var wire editableProjectWire
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Handoff) == 0 || len(wire.Handoff) > 16000 {
		t.Fatalf("handoff missing or unbounded: %d bytes", len(wire.Handoff))
	}
	var excerpt struct {
		Truncated bool `json:"truncated"`
	}
	if err = json.Unmarshal(wire.Handoff, &excerpt); err != nil {
		t.Fatal(err)
	}
	if !excerpt.Truncated {
		t.Fatal("shortened handoff excerpts were presented as complete records")
	}
	saved, err := e.Store.Get(task.ID)
	if err != nil || len(saved.Audit.Facts[0]) < 10000 {
		t.Fatalf("bounded handoff discarded raw source record: %v", err)
	}
}

func TestProjectHandoffUsesAcceptedCommitForNewUserMemory(t *testing.T) {
	e, task, h, before := editableProjectFixture(t)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.State, saved.Base = stateAccepted, "before-acceptance"
		saved.LocalCheckpoint = &AcceptanceCheckpoint{Complete: true, Commit: "accepted-source"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	code, wire, raw := projectEditRequest(t, h, task, map[string]any{
		"expected_revision": before.Revision,
		"memory":            map[string]string{"kind": "constraint", "text": "Keep the accepted interface contract"},
	})
	if code != http.StatusOK {
		t.Fatalf("save current memory: %d %s", code, raw)
	}
	var handoff struct {
		SourceCommit string   `json:"source_commit"`
		Warnings     []string `json:"warnings"`
	}
	if err = json.Unmarshal(wire.Handoff, &handoff); err != nil {
		t.Fatal(err)
	}
	if handoff.SourceCommit != "accepted-source" {
		t.Fatalf("handoff used pre-acceptance version: %+v", handoff)
	}
	for _, warning := range handoff.Warnings {
		if strings.Contains(warning, "部分项目记忆关联旧版本") {
			t.Fatal("new memory was falsely labeled stale against accepted source")
		}
	}
}
