package reliable

import (
	"os"
	"testing"
)

// Explicit, read-only deployment proof against a closed copy of production.
// This authenticates legacy human acceptance without running models, applying
// patches, changing the original Git repository, or rescanning its resources.
func TestExistingAcceptedCheckpointEvidenceRemainsValid(t *testing.T) {
	id := os.Getenv("XIAOBUDING_ACCEPTED_PARENT")
	db := os.Getenv("XIAOBUDING_LEGACY_COPY_DB")
	root := os.Getenv("XIAOBUDING_LEGACY_EVIDENCE_ROOT")
	if id == "" || db == "" || root == "" {
		t.Skip("explicit accepted checkpoint deployment verification is disabled")
	}
	store, err := OpenStore(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	e := &Engine{Store: store, Root: root}
	task, err := store.Get(id)
	if err != nil || task.State != stateAccepted {
		t.Fatalf("expected an existing human-accepted parent: %v", err)
	}
	if err = e.acceptanceReceipt(task, false); err != nil {
		t.Fatal(err)
	}
	if _, err = e.frozenAcceptedPatch(task); err != nil {
		t.Fatal(err)
	}
	t.Logf("verified original human receipt and frozen seal for %s, round %d, patch %s", id, task.Round, task.PatchHash)
}
