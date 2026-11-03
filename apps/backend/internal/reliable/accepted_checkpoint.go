package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func acceptancePending(t *Task) bool {
	return t.LocalCheckpoint != nil && t.State != stateAccepted
}

func (e *Engine) Accept(ctx context.Context, id string) error {
	ctx, release, err := e.reserve(ctx, id)
	if err != nil {
		return err
	}
	defer release()
	ctx = observeActivity(ctx, func(event Activity) { e.appendActivity(id, event) })
	t, err := e.Store.Get(id)
	if err != nil {
		return err
	}
	if t.ArchivedAt != "" {
		return errors.New("task_archived: restore the task before accepting it")
	}
	if t.State == stateAccepted && t.LocalCheckpoint != nil && t.LocalCheckpoint.Complete {
		return nil
	}
	if !acceptancePending(t) {
		if t.State != stateAwaitingAcceptance {
			return errors.New("this task is not ready for human acceptance")
		}
		if _, err = e.acceptanceEvidence(ctx, t); err == nil {
			t, err = e.beginAcceptance(t)
		}
		if err != nil {
			_ = e.setState(id, stateBlocked, err.Error())
			return err
		}
	}
	err = e.finishAcceptance(ctx, t)
	e.recordRunFailure(id, err)
	return err
}

func (e *Engine) beginAcceptance(t *Task) (*Task, error) {
	return e.Store.Update(t.ID, func(saved *Task) error {
		if saved.State != stateAwaitingAcceptance || saved.EvidenceHash != t.EvidenceHash || saved.RequirementRevision != saved.AuditedRevision {
			return errors.New("acceptance_changed: the reviewed result changed before acceptance")
		}
		saved.LocalCheckpoint = &AcceptanceCheckpoint{Base: saved.Base}
		saved.State, saved.Reason = stateApplying, "Human explicitly accepted; applying and saving the verified local result"
		return nil
	})
}

func (e *Engine) finishAcceptance(ctx context.Context, t *Task) error {
	patch, err := e.frozenAcceptedPatch(t)
	if err != nil {
		return err
	}
	if err = e.setState(t.ID, stateApplying, "Completing the previously accepted local checkpoint"); err != nil {
		return err
	}
	emitActivity(ctx, "phase", "保存验收成果", "只将已验收的修改记入本地 Git，不上传，不复制整个项目。")
	if err = e.checkpointAcceptedTask(ctx, t, patch, true); err != nil {
		return err
	}
	if err = e.acceptanceReceipt(t, true); err != nil {
		return err
	}
	_, err = e.Store.Update(t.ID, func(saved *Task) error {
		if saved.LocalCheckpoint == nil || !saved.LocalCheckpoint.Complete {
			return errors.New("local_checkpoint_pending: local checkpoint has not completed")
		}
		saved.State, saved.Reason = stateAccepted, "Human accepted; verified changes applied and checkpointed in local Git"
		return nil
	})
	return err
}

func (e *Engine) checkpointAcceptedTask(ctx context.Context, t *Task, patch string, allowApply bool) error {
	persist := func(point *AcceptanceCheckpoint) error {
		_, err := e.Store.Update(t.ID, func(saved *Task) error {
			if saved.Base != t.Base || saved.PatchHash != t.PatchHash || saved.EvidenceHash != t.EvidenceHash {
				return errors.New("acceptance_changed: checkpoint identity changed")
			}
			saved.LocalCheckpoint = point
			return nil
		})
		return err
	}
	_, err := checkpointAcceptedPatch(ctx, t.Project, t.Base, patch, filepath.Join(e.Root, "evidence", t.ID), t.LocalCheckpoint, persist, allowApply)
	return err
}

// Legacy acceptance already authorized this immutable patch. Verify the saved
// receipt and seal, without rescanning the old mutable candidate's resource tree.
func (e *Engine) recoverAcceptedParent(ctx context.Context, child *Task) error {
	if child.ParentTaskID == "" {
		return nil
	}
	parent, err := e.Store.Get(child.ParentTaskID)
	if err != nil {
		return err
	}
	if parent.State != stateAccepted || projectKey(parent.Project) != projectKey(child.Project) {
		return nil
	}
	if parent.LocalCheckpoint != nil && parent.LocalCheckpoint.Complete {
		return nil
	}
	if parent.LocalCheckpoint == nil && requireClean(ctx, child.Project, "") == nil {
		return nil // A user's own completed commit is also a valid next-batch base.
	}
	if err = e.acceptanceReceipt(parent, false); err != nil {
		return err
	}
	patch, err := e.frozenAcceptedPatch(parent)
	if err != nil {
		return err
	}
	emitActivity(ctx, "phase", "补齐上一批本地存档", "核对已验收补丁，复用现有 Git 历史；不增加修改轮次。")
	return e.checkpointAcceptedTask(ctx, parent, patch, false)
}

func (e *Engine) frozenAcceptedPatch(t *Task) (string, error) {
	if t.RequirementRevision != t.AuditedRevision {
		return "", errRequirementsChanged
	}
	hash, err := e.evidenceDigest(t)
	if err != nil || hash == "" || hash != t.EvidenceHash {
		return "", errors.New("local_checkpoint_evidence: saved evidence no longer matches the accepted result")
	}
	patch := filepath.Join(e.Root, "evidence", t.ID, fmt.Sprintf("round-%d", t.Round), "changes.patch")
	raw, err := os.ReadFile(patch)
	if err != nil {
		return "", err
	}
	if digest(raw) != t.PatchHash {
		return "", errors.New("local_checkpoint_evidence: saved patch no longer matches the accepted result")
	}
	return patch, nil
}

func (e *Engine) acceptanceReceipt(t *Task, create bool) error {
	want := map[string]any{"source_hash": t.Snapshot, "patch_hash": t.PatchHash, "round": t.Round, "human_action": "accept_and_apply"}
	path := filepath.Join(e.Root, "evidence", t.ID, "acceptance.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && create {
		return e.saveJSON(t.ID, "acceptance.json", want)
	}
	if err != nil {
		return fmt.Errorf("local_checkpoint_receipt: %w", err)
	}
	var got map[string]any
	expected, _ := json.Marshal(want)
	if json.Unmarshal(raw, &got) != nil {
		return errors.New("local_checkpoint_receipt: original human acceptance receipt does not match")
	}
	actual, _ := json.Marshal(got)
	if !bytes.Equal(actual, expected) {
		return errors.New("local_checkpoint_receipt: original human acceptance receipt does not match")
	}
	return nil
}
