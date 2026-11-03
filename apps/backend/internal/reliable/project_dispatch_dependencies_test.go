package reliable

import (
	"strings"
	"testing"
)

func TestProjectDependencyCommitRecheckRollsBackBatchCreation(t *testing.T) {
	e, parent, h, before := editableProjectFixture(t)
	dependency, childID := before.Items[0].ID, before.Items[1].ID
	code, _, raw := projectEditRequest(t, h, parent, map[string]any{"expected_revision": before.Revision, "item_edit": map[string]any{"id": childID, "depends_on": []string{dependency}}})
	if code != 200 {
		t.Fatalf("dependency setup: %d %s", code, raw)
	}
	if _, err := e.Store.Update(parent.ID, func(saved *Task) error { saved.State = stateAccepted; return nil }); err != nil {
		t.Fatal(err)
	}
	record, err := e.Project(parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	item := record.Items[1]
	child := &Task{ID: "dependency-child", Project: parent.Project, Goal: item.Title, State: stateQueued}
	message := &TaskMessage{ID: "project-" + digest([]byte(item.ID))[:26], Content: item.Title, LinkedTask: child.ID, Status: "linked"}
	err = e.Store.createLinked(parent.ID, child, message)
	if err == nil || !strings.Contains(err.Error(), "project_dependency_pending") {
		t.Fatalf("commit did not recheck dependency: %v", err)
	}
	if _, err = e.Store.Get(child.ID); err == nil {
		t.Fatal("rejected binding left a child task behind")
	}
	saved, err := e.Store.Get(parent.ID)
	if err != nil || len(saved.Messages) != 0 {
		t.Fatalf("rejected binding mutated parent messages: %v", err)
	}
	_, err = e.Store.updateProject(parent.Project, -1, func(ledger *ProjectRecord) error { ledger.Items[0].TaskID = parent.ID; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Store.createLinked(parent.ID, child, message); err != nil {
		t.Fatalf("accepted positive control cannot commit: %v", err)
	}
	record, err = e.Project(parent.ID)
	if err != nil || record.Items[1].TaskID != child.ID {
		t.Fatalf("valid dependency did not bind atomically: %v", err)
	}
}

func TestProjectDependencyAutomaticNextSkipsUnacceptedPrerequisite(t *testing.T) {
	e, parent, h, before := editableProjectFixture(t)
	e.Providers = testProviders()
	t.Cleanup(e.wg.Wait)
	dependency, childID := before.Items[0].ID, before.Items[1].ID
	code, edited, raw := projectEditRequest(t, h, parent, map[string]any{
		"expected_revision": before.Revision,
		"item_edit":         map[string]any{"id": childID, "depends_on": []string{dependency}},
		"item_order":        []string{childID, dependency},
	})
	if code != 200 {
		t.Fatalf("ordered dependency setup: %d %s", code, raw)
	}
	if _, err := e.Store.Update(parent.ID, func(saved *Task) error { saved.State = stateAccepted; return nil }); err != nil {
		t.Fatal(err)
	}
	code, _, raw = projectEditRequest(t, h, parent, map[string]any{"expected_revision": edited.Revision, "auto_continue": true})
	if code != 200 {
		t.Fatalf("enable automatic next: %d %s", code, raw)
	}
	e.wg.Wait()
	record, err := e.Project(parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Items[0].TaskID != "" || record.Items[1].TaskID == "" {
		t.Fatalf("automatic next bypassed prerequisite: %+v", record.Items)
	}
	started, err := e.Store.Get(record.Items[1].TaskID)
	if err != nil || started.State == stateAccepted {
		t.Fatalf("prerequisite was automatically accepted: %v", err)
	}
}
