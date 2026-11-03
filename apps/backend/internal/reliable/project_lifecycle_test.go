package reliable

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectPlanRequiresConfirmationBeforeAutomaticContinuation(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	e.Providers = testProviders()
	t.Cleanup(e.wg.Wait)
	if _, err := e.Store.Update(task.ID, func(saved *Task) error { saved.State = stateAccepted; return nil }); err != nil {
		t.Fatal(err)
	}
	message := &TaskMessage{ID: "project-plan-fixture"}
	if err := e.recordProjectPlan(task, message, &Audit{Plan: []string{"Add a CSV export with an integration check"}, Facts: []string{"app.txt is the current input"}}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	record, err := e.EditProject(task.ID, projectEdit{ExpectedRevision: 1, AutoContinue: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	tasks, _ := e.Store.List()
	if len(tasks) != 1 || record.Items[0].Confirmed || record.Memories[0].Verified {
		t.Fatal("unconfirmed model suggestions started work or became verified facts")
	}
	started, err := e.StartProjectItem(task.ID, record.Items[0].ID)
	if err != nil || started.LinkedTask == "" {
		t.Fatalf("explicitly confirming a proposal could not start its batch: %v", err)
	}
	again, err := e.StartProjectItem(task.ID, record.Items[0].ID)
	if err != nil || again.LinkedTask != started.LinkedTask {
		t.Fatalf("retry created a second batch or lost the assignment: %v", err)
	}
	e.wg.Wait()
	record, _ = e.Project(started.LinkedTask)
	if !record.Items[0].Confirmed || record.Items[0].TaskID != started.LinkedTask || record.ID != projectLedgerID(task.Project) {
		t.Fatal("project requirements did not follow the linked batch")
	}
}

func TestProjectUserBacklogStartsOnlyAfterAcceptance(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	e.Providers = testProviders()
	t.Cleanup(e.wg.Wait)
	enabled := true
	_, err := e.EditProject(task.ID, projectEdit{ExpectedRevision: 0, AutoContinue: &enabled, Items: []ProjectItem{{Title: "Continue the accepted implementation", Acceptance: "Preserve the existing behavior"}}})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := e.Store.List()
	if len(before) != 1 {
		t.Fatal("backlog started before acceptance")
	}
	if err = e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	waiting, _ := e.Store.List()
	if len(waiting) != 1 {
		t.Fatal("a passing review bypassed human acceptance")
	}
	if err = e.Accept(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	after, _ := e.Store.List()
	record, _ := e.Project(task.ID)
	if len(after) != 2 || record.Items[0].TaskID == "" || record.Items[0].Status == stateAccepted {
		t.Fatal("human acceptance did not advance the confirmed backlog exactly once")
	}
}

func TestProjectCannotForgeProofOrClaimStaleChecksPassed(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "project-security"}
	url := "/api/v1/reliable/tasks/" + task.ID + "/project"
	for _, body := range []string{
		`{"expected_revision":0,"items":[{"title":"Fake","status":"accepted"}]}`,
		`{"expected_revision":0,"items":[{"title":"Fake","acceptance":"bad\u0000text"}]}`,
		`{"expected_revision":0,"memory":{"kind":"constraint","text":"Claim","verified":true}}`,
	} {
		if response := httpRequest(h, http.MethodPost, url, body, h.Token); response.Code < 400 {
			t.Fatalf("forged evidence was accepted: %s", body)
		}
	}
	if _, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.State = stateAwaitingAcceptance
		saved.RequirementRevision, saved.AuditedRevision = 2, 1
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	record, _ := e.Project(task.ID)
	if record.Items[0].Status == "verified" {
		t.Fatal("a stale or missing verification was labeled as verified")
	}
}

func TestProjectFullMemoryIsSavedWithDispatchEvidence(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	_, err := e.EditProject(task.ID, projectEdit{ExpectedRevision: 0, Memory: &ProjectMemory{Kind: "constraint", Text: "Preserve all original question resources"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	saved, _ := e.Store.Get(task.ID)
	var input handoffInput
	raw, err := os.ReadFile(filepath.Join(e.Root, "evidence", task.ID, saved.Runs[0].InputEvidence))
	if err != nil || json.Unmarshal(raw, &input) != nil || input.ProjectRecord == nil || len(input.ProjectRecord.Memories) != 1 {
		t.Fatalf("full project context was not retained in immutable dispatch evidence: %v", err)
	}
}

func TestProjectPlanningContextRemainsBoundedWithoutDiscardingStoredNotes(t *testing.T) {
	record := emptyProject("E:/project")
	for i := 0; i < 20; i++ {
		record.Items = append(record.Items, ProjectItem{Title: strings.Repeat("长期项目验收条件", 120), Acceptance: strings.Repeat("保留原始数据", 120)})
		record.Memories = append(record.Memories, ProjectMemory{Text: strings.Repeat("\\\"记住约定", 120), Kind: "constraint"})
	}
	context := compactProjectContext(record)
	if len(context) > 9500 || len(record.Items) != 20 || len(record.Memories) != 20 {
		t.Fatal("planning exceeded the message budget or removed persistent knowledge")
	}
}
