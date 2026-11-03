package reliable

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type recoveryReceipt struct {
	TaskID              string `json:"task_id"`
	Attempt             string `json:"attempt"`
	Project             string `json:"project"`
	Worktree            string `json:"worktree"`
	Base                string `json:"base"`
	Round               int    `json:"round"`
	RequirementRevision int    `json:"requirement_revision"`
	PlanRevision        int    `json:"plan_revision"`
	SourceHash          string `json:"source_hash"`
	ResourceHash        string `json:"resource_hash,omitempty"`
	PreviousReason      string `json:"previous_reason,omitempty"`
	Time                string `json:"time"`
}

func (e *Engine) RecoverCancelled(id string) (*Task, error) {
	e.mu.Lock()
	ctx, release, err := e.reserveLocked(context.Background(), id)
	if err != nil {
		e.mu.Unlock()
		return nil, fmt.Errorf("recovery_active: %w", err)
	}
	attempt := &RecoveryAttempt{ID: randomID()}
	attempt.Intent = "recovery-" + attempt.ID + "-intent.json"
	task, err := e.Store.claimCancelled(id, attempt)
	e.mu.Unlock()
	if err != nil {
		release()
		return nil, err
	}
	if err = e.saveJSON(id, attempt.Intent, recoveryRecord(task, attempt.ID, nil)); err != nil {
		e.failRecovery(id, fmt.Errorf("recovery_evidence_invalid: %w", err))
		release()
		return nil, err
	}
	go func() {
		defer release()
		if runErr := e.runTask(ctx, id); runErr != nil {
			e.recordRunFailure(id, runErr)
		}
	}()
	return task, nil
}

func recoveryRecord(t *Task, attempt string, snap *Snapshot) recoveryReceipt {
	r := recoveryReceipt{TaskID: t.ID, Attempt: attempt, Project: t.Project, Worktree: t.Worktree, Base: t.Base, Round: t.Round, RequirementRevision: t.RequirementRevision, PlanRevision: t.PlanRevision, Time: time.Now().UTC().Format(time.RFC3339Nano)}
	if t.RecoveryPending != nil {
		r.PreviousReason = t.RecoveryPending.PreviousReason
	}
	if snap != nil {
		r.SourceHash, r.ResourceHash = snap.Hash, snap.ResourceHash
	}
	return r
}

func (e *Engine) beginRecoveryValidation(t *Task) error {
	if t.RecoveryPending != nil {
		return nil
	}
	attempt := &RecoveryAttempt{ID: randomID(), Round: t.Round, PreviousReason: t.Reason}
	attempt.Intent = "recovery-" + attempt.ID + "-intent.json"
	if err := e.saveJSON(t.ID, attempt.Intent, recoveryRecord(t, attempt.ID, nil)); err != nil {
		return fmt.Errorf("recovery_evidence_invalid: %w", err)
	}
	updated, err := e.Store.Update(t.ID, func(saved *Task) error {
		if saved.State == stateCancelled || saved.State == stateCancelling {
			return context.Canceled
		}
		saved.RecoveryPending = attempt
		return nil
	})
	if err == nil {
		*t = *updated
	}
	return err
}

func (e *Engine) validateRecovery(ctx context.Context, t *Task) error {
	if err := e.recoveryQuiescence(t); err != nil {
		return err
	}
	if err := e.validateRecoveryHistory(t); err != nil {
		return fmt.Errorf("recovery_evidence_invalid: %w", err)
	}
	if t.Worktree == "" {
		if err := e.recoverPreparation(ctx, t); err != nil {
			return err
		}
	}
	snap, err := e.validateRecoveryCandidate(ctx, t)
	if err != nil {
		return err
	}
	return e.publishRecovery(ctx, t, snap)
}

func (e *Engine) recoveryQuiescence(t *Task) error {
	if err := e.processQuiescent(t); err != nil {
		return err
	}
	reason := t.Reason
	if t.RecoveryPending != nil {
		reason += t.RecoveryPending.PreviousReason
	}
	if !recoveryProcessProofRequired(reason) {
		return nil
	}
	name, want, err := quiescenceReceipt(t)
	if err == nil {
		err = e.readQuiescenceReceipt(name, want)
	}
	if err != nil {
		return fmt.Errorf("recovery_process_unknown: previous process cleanup failed: %w", err)
	}
	return nil
}

func recoveryProcessProofRequired(reason string) bool {
	for _, marker := range []string{"recovery_process_unknown", "owned_process_cleanup", "TerminateProcess", "wait for job processes"} {
		if strings.Contains(reason, marker) {
			return true
		}
	}
	return false
}

func (e *Engine) publishRecovery(ctx context.Context, t *Task, snap *Snapshot) error {
	attempt := t.RecoveryPending
	if attempt == nil {
		attempt = &RecoveryAttempt{ID: randomID(), Round: t.Round}
	}
	name := "recovery-" + attempt.ID + "-" + randomID() + "-validated.json"
	if err := e.saveJSON(t.ID, name, recoveryRecord(t, attempt.ID, snap)); err != nil {
		return fmt.Errorf("recovery_evidence_invalid: %w", err)
	}
	updated, err := e.Store.Update(t.ID, func(saved *Task) error {
		if ctx.Err() != nil || saved.State == stateCancelled || saved.State == stateCancelling {
			return context.Canceled
		}
		if saved.Round != t.Round || saved.RequirementRevision != t.RequirementRevision || saved.PlanRevision != t.PlanRevision {
			return errors.New("recovery_evidence_invalid: task changed during validation")
		}
		if err := e.promoteFuturePlan(saved); err != nil {
			return err
		}
		if saved.RecoveryPending != nil {
			saved.RecoveryEvidence = append(saved.RecoveryEvidence, saved.RecoveryPending.Intent)
		}
		saved.RecoveryEvidence = append(saved.RecoveryEvidence, name)
		saved.RecoveryPending = nil
		saved.Snapshot, saved.EvidenceHash, saved.PatchHash = "", "", ""
		saved.Review, saved.Checks, saved.RequiredReads = nil, nil, nil
		recoverDispatches(saved)
		saved.Reason = "Retained candidate validated; starting a fresh audit with the remaining round budget"
		return nil
	})
	if err == nil {
		*t = *updated
	}
	return err
}
