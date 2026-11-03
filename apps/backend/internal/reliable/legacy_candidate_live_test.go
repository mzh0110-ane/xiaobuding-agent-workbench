package reliable

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestExistingLegacySealedCandidateRemainsValid(t *testing.T) {
	db, root := os.Getenv("XIAOBUDING_LEGACY_COPY_DB"), os.Getenv("XIAOBUDING_LEGACY_EVIDENCE_ROOT")
	if db == "" || root == "" {
		t.Skip("explicit legacy candidate verification is disabled")
	}
	store, err := OpenStore(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	engine := &Engine{Store: store, Root: root}
	tasks, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	verifyDeploymentSnapshot(t, tasks)
	checked := 0
	for _, task := range tasks {
		if task.State != stateAwaitingAcceptance {
			continue
		}
		if _, err = engine.acceptanceEvidence(context.Background(), task); err != nil {
			t.Fatalf("legacy sealed candidate %s no longer validates: %v", task.ID, err)
		}
		checked++
	}
	t.Logf("verified %d existing candidates without applying their patches", checked)
}

func verifyDeploymentSnapshot(t *testing.T, tasks []*Task) {
	t.Helper()
	path := os.Getenv("XIAOBUDING_EXPECTED_TASKS_JSON")
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var expected []*Task
	if err = json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != len(expected) {
		t.Fatal("task history changed before deployment")
	}
	byID := make(map[string]*Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}
	for _, old := range expected {
		got := byID[old.ID]
		if got == nil || got.State != old.State || got.Round != old.Round || got.EvidenceHash != old.EvidenceHash || got.ArchivedAt != old.ArchivedAt || got.Updated != old.Updated {
			t.Fatalf("task %s changed between the online snapshot and shutdown", old.ID)
		}
	}
	t.Logf("verified %d stopped tasks against the online snapshot", len(tasks))
}
