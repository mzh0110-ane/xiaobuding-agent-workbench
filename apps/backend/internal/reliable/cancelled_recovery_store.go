package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
)

type RecoveryAttempt struct {
	ID             string `json:"id"`
	Intent         string `json:"intent"`
	Round          int    `json:"round"`
	PreviousReason string `json:"previous_reason,omitempty"`
}

func recoveryEligible(t *Task) error {
	if t.State != stateCancelled || t.ReadOnly || t.ArchivedAt != "" {
		return errors.New("recovery_ineligible: restore an unarchived stopped modification task first")
	}
	if t.Round >= MaxRounds {
		return errors.New("recovery_exhausted: this batch has used all three implementation rounds")
	}
	return nil
}

func (s *Store) claimCancelled(id string, attempt *RecoveryAttempt) (*Task, error) {
	var task Task
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(tasksBucket)
		if err := json.Unmarshal(bucket.Get([]byte(id)), &task); err != nil {
			return err
		}
		if err := recoveryEligible(&task); err != nil {
			return err
		}
		if err := bucket.ForEach(func(_, raw []byte) error {
			var other Task
			if err := json.Unmarshal(raw, &other); err != nil {
				return err
			}
			if other.ID != id && projectsOverlap(task.Project, other.Project) && other.State != stateAccepted && other.State != stateCancelled && other.State != stateInspected && !supersededProjectTask(tx, &other) {
				return errors.New("recovery_project_conflict: another unfinished task owns this project")
			}
			return nil
		}); err != nil {
			return err
		}
		attempt.Round = task.Round
		attempt.PreviousReason = task.Reason
		if task.RecoveryPending != nil && recoveryProcessProofRequired(task.RecoveryPending.PreviousReason) {
			attempt.PreviousReason = task.RecoveryPending.PreviousReason
		}
		task.RecoveryPending, task.State = attempt, stateQueued
		task.ContinuePending = false
		task.Reason = "Recovery requested; validating the retained candidate before dispatch"
		task.Updated = time.Now().UTC().Format(time.RFC3339Nano)
		raw, err := json.Marshal(&task)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(id), raw)
	})
	return &task, err
}

func (e *Engine) failRecovery(id string, runErr error) {
	_, _ = e.Store.Update(id, func(t *Task) error {
		if t.State == stateAccepted || t.State == stateApplying {
			return context.Canceled
		}
		t.State = stateCancelled
		t.Reason = runErr.Error()
		return nil
	})
}
