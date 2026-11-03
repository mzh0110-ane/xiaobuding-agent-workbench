package reliable

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func historicalReplanFixture(t *testing.T, approvals int) (*Engine, *Task, string) {
	t.Helper()
	root := t.TempDir()
	store, err := OpenStore(filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	engine := &Engine{Store: store, Root: root}
	task := &Task{ID: "receipt-history", Project: t.TempDir(), Worktree: t.TempDir(), Base: "original-base", Goal: "Keep valid receipt provenance", State: stateBlocked, AllowedPaths: []string{"src"}}
	if err = store.Create(task); err != nil {
		t.Fatal(err)
	}
	first := ""
	for index := 0; index < approvals; index++ {
		approval, saveErr := engine.saveAuditReplanApproval(context.Background(), task, auditReplanRequest{SourceHash: digest([]byte(fmt.Sprint(index))), Confirm: true, Note: "Original human note"})
		if saveErr != nil {
			t.Fatal(saveErr)
		}
		if first == "" {
			first = approval.Receipt
		}
		task, err = store.Update(task.ID, func(saved *Task) error {
			saved.Round++
			saved.RequirementRevision++
			saved.AuditedRevision = saved.RequirementRevision
			saved.PlanRevision++
			saved.State = stateBlocked
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	task, err = store.Update(task.ID, func(saved *Task) error { saved.State = stateReviewing; return nil })
	if err != nil {
		t.Fatal(err)
	}
	round := fmt.Sprintf("round-%d", task.Round)
	for _, name := range []string{"contract.json", round + "-audit.json", round + "-before.json", round + "-implementation.json", round + "-checks.json", round + "-source.json", round + "-review.json", round + "-decision.json"} {
		if err = engine.saveJSON(task.ID, name, map[string]bool{"fixture": true}); err != nil {
			t.Fatal(err)
		}
	}
	patch := filepath.Join(root, "evidence", task.ID, round, "changes.patch")
	if err = os.MkdirAll(filepath.Dir(patch), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(patch, []byte("verified fixture patch\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return engine, task, first
}

func rewriteHumanReceipt(t *testing.T, engine *Engine, task *Task, name string) {
	t.Helper()
	file := filepath.Join(engine.Root, "evidence", task.ID, name)
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err = json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	receipt["note"] = "Model changed this valid JSON before review sealing"
	raw, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestAuditReplanEvidenceSealingRejectsReceiptTamperAfterRoundAdvance(t *testing.T) {
	engine, task, first := historicalReplanFixture(t, 1)
	if hash, err := engine.evidenceDigestSnapshot(task); err != nil || hash == "" {
		t.Fatalf("later legitimate round/revisions must remain valid: %s %v", hash, err)
	}
	rewriteHumanReceipt(t, engine, task, first)
	if err := engine.sealReview(task.ID, Pass, "Independent review passed", task.RequirementRevision); err == nil {
		t.Fatal("valid JSON tampering was accepted as a new human approval during sealing")
	}
	saved, err := engine.Store.Get(task.ID)
	if err != nil || saved.State != stateReviewing || saved.EvidenceHash != "" {
		t.Fatalf("rejected sealing changed task state: %v %+v", err, saved)
	}
}

func TestAuditReplanEvidenceAnchorsEarlierApprovalsAcrossLaterRequirements(t *testing.T) {
	engine, task, first := historicalReplanFixture(t, 2)
	if hash, err := engine.evidenceDigestSnapshot(task); err != nil || hash == "" {
		t.Fatalf("valid historical approvals were rejected: %s %v", hash, err)
	}
	rewriteHumanReceipt(t, engine, task, first)
	if _, err := engine.evidenceDigestSnapshot(task); err == nil {
		t.Fatal("overwriting the latest approval pointer made an earlier human receipt mutable")
	}
}
