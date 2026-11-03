package reliable

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type editableProjectWire struct {
	Revision int `json:"revision"`
	Items    []struct {
		ID, Title, Status, Acceptance string
		ShortTitle                    string   `json:"short_title"`
		Milestone                     string   `json:"milestone"`
		DependsOn                     []string `json:"depends_on"`
		Reason                        string   `json:"status_reason"`
	} `json:"items"`
	Handoff json.RawMessage `json:"handoff"`
}

func projectEditRequest(t *testing.T, h *HTTP, task *Task, body map[string]any) (int, editableProjectWire, string) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	response := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+task.ID+"/project", string(raw), h.Token)
	var wire editableProjectWire
	if response.Code == http.StatusOK {
		if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil {
			t.Fatal(err)
		}
	}
	return response.Code, wire, response.Body.String()
}

func editableProjectFixture(t *testing.T) (*Engine, *Task, *HTTP, editableProjectWire) {
	t.Helper()
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "project-edit-fixture"}
	code, wire, raw := projectEditRequest(t, h, task, map[string]any{
		"expected_revision": 0,
		"items":             []map[string]string{{"title": "Keep this complete original requirement and its detailed scope", "acceptance": "Original acceptance"}, {"title": "Second requirement"}},
	})
	if code != http.StatusOK {
		t.Fatalf("seed project: %d %s", code, raw)
	}
	return e, task, h, wire
}

func TestProjectBacklogEditsRetainGoalAndStableOrder(t *testing.T) {
	e, task, h, before := editableProjectFixture(t)
	a, b := before.Items[0].ID, before.Items[1].ID
	code, after, raw := projectEditRequest(t, h, task, map[string]any{
		"expected_revision": before.Revision,
		"item_edit":         map[string]any{"id": a, "short_title": "Readable label", "milestone": "First release", "depends_on": []string{b}},
		"item_order":        []string{b, a},
	})
	if code != http.StatusOK {
		t.Fatalf("backlog edit is unavailable: %d %s", code, raw)
	}
	if after.Revision != before.Revision+1 || after.Items[0].ID != b || after.Items[1].ID != a {
		t.Fatalf("edit/order was not atomic: %+v", after)
	}
	got := after.Items[1]
	if got.Title != before.Items[0].Title || got.Acceptance != "Original acceptance" || got.ShortTitle != "Readable label" || got.Milestone != "First release" || len(got.DependsOn) != 1 || got.DependsOn[0] != b {
		t.Fatalf("short label erased requirement or metadata was not retained: %+v", got)
	}
	persisted, err := e.Project(task.ID)
	if err != nil || persisted.Items[0].ID != b {
		t.Fatalf("order did not survive a fresh read: %v", err)
	}
	code, _, _ = projectEditRequest(t, h, task, map[string]any{"expected_revision": before.Revision, "item_edit": map[string]any{"id": a, "short_title": "Stale write"}})
	if code < 400 {
		t.Fatal("stale revision overwrote current project")
	}
}

func TestProjectBacklogDispositionRequiresReasonAndCanResume(t *testing.T) {
	_, task, h, before := editableProjectFixture(t)
	id := before.Items[0].ID
	for _, state := range []string{"deferred", "cancelled"} {
		code, _, _ := projectEditRequest(t, h, task, map[string]any{"expected_revision": before.Revision, "item_edit": map[string]any{"id": id, "disposition": state}})
		if code < 400 {
			t.Fatalf("%s silently dropped a requirement without reason", state)
		}
	}
	code, deferred, raw := projectEditRequest(t, h, task, map[string]any{"expected_revision": before.Revision, "item_edit": map[string]any{"id": id, "disposition": "deferred", "reason": "Waiting for sample data"}})
	if code != 200 || deferred.Items[0].Status != "deferred" || deferred.Items[0].Reason != "Waiting for sample data" {
		t.Fatalf("cannot defer transparently: %d %s", code, raw)
	}
	code, resumed, raw := projectEditRequest(t, h, task, map[string]any{"expected_revision": deferred.Revision, "item_edit": map[string]any{"id": id, "disposition": "planned"}})
	if code != 200 || resumed.Items[0].Status != projectPlanned || resumed.Items[0].Reason != "" {
		t.Fatalf("cannot restore pending requirement: %d %s", code, raw)
	}
}

func TestProjectBacklogRejectsBadDependenciesAndForgedCompletion(t *testing.T) {
	e, task, h, before := editableProjectFixture(t)
	a, b := before.Items[0].ID, before.Items[1].ID
	cases := []map[string]any{
		{"id": a, "depends_on": []string{a}},
		{"id": a, "depends_on": []string{"missing"}},
		{"id": a, "depends_on": []string{b, b}},
		{"id": a, "disposition": "accepted"},
		{"id": a, "disposition": "verified"},
		{"id": a, "short_title": "bad\x00title"},
		{"id": a, "status": "accepted"},
	}
	for _, edit := range cases {
		code, _, _ := projectEditRequest(t, h, task, map[string]any{"expected_revision": before.Revision, "item_edit": edit})
		if code < 400 {
			t.Fatalf("unsafe project mutation accepted: %+v", edit)
		}
	}
	for _, order := range [][]string{{a, a}, {a, "missing"}, {a}} {
		code, _, _ := projectEditRequest(t, h, task, map[string]any{"expected_revision": before.Revision, "item_order": order})
		if code < 400 {
			t.Fatalf("incomplete/duplicate ordering accepted: %+v", order)
		}
	}
	after, err := e.Project(task.ID)
	if err != nil || after.Revision != before.Revision {
		t.Fatalf("rejected mutation changed ledger: %v", err)
	}
	code, linked, raw := projectEditRequest(t, h, task, map[string]any{"expected_revision": before.Revision, "item_edit": map[string]any{"id": a, "depends_on": []string{b}}})
	if code != 200 {
		t.Fatalf("valid dependency rejected: %d %s", code, raw)
	}
	code, _, _ = projectEditRequest(t, h, task, map[string]any{"expected_revision": linked.Revision, "item_edit": map[string]any{"id": b, "depends_on": []string{a}}})
	if code < 400 {
		t.Fatal("dependency cycle accepted")
	}
}

func TestProjectBacklogCannotEditAssignedItems(t *testing.T) {
	e, task, h, before := editableProjectFixture(t)
	_, err := e.Store.updateProject(task.Project, before.Revision, func(record *ProjectRecord) error {
		record.Items[0].TaskID = task.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{stateImplementing, stateAwaitingAcceptance, stateAccepted} {
		if _, err = e.Store.Update(task.ID, func(saved *Task) error { saved.State = state; return nil }); err != nil {
			t.Fatal(err)
		}
		code, _, _ := projectEditRequest(t, h, task, map[string]any{"expected_revision": before.Revision + 1, "item_edit": map[string]any{"id": before.Items[0].ID, "title": "Changed requirement"}})
		if code < 400 {
			t.Fatalf("assigned %s requirement changed under existing proof", state)
		}
	}
}

func TestProjectDependencyRequiresHumanAcceptanceBeforeDispatch(t *testing.T) {
	e, task, h, before := editableProjectFixture(t)
	e.Providers = testProviders()
	t.Cleanup(e.wg.Wait)
	dependency, child := before.Items[0].ID, before.Items[1].ID
	code, _, raw := projectEditRequest(t, h, task, map[string]any{"expected_revision": before.Revision, "item_edit": map[string]any{"id": child, "depends_on": []string{dependency}}})
	if code != 200 {
		t.Fatalf("dependency setup: %d %s", code, raw)
	}
	if _, err := e.Store.Update(task.ID, func(saved *Task) error { saved.State = stateAccepted; return nil }); err != nil {
		t.Fatal(err)
	}
	_, err := e.StartProjectItem(task.ID, child)
	if err == nil || !strings.Contains(err.Error(), "project_dependency_pending") {
		t.Fatalf("pending dependency did not block dispatch explicitly: %v", err)
	}
	items, err := e.Store.List()
	if err != nil || len(items) != 1 {
		t.Fatalf("blocked dependency spawned a batch: %v", err)
	}
	_, err = e.Store.updateProject(task.Project, -1, func(record *ProjectRecord) error { record.Items[0].TaskID = task.ID; return nil })
	if err != nil {
		t.Fatal(err)
	}
	started, err := e.StartProjectItem(task.ID, child)
	if err != nil || started.LinkedTask == "" {
		t.Fatalf("accepted dependency still blocked dispatch: %v", err)
	}
	e.wg.Wait()
}
