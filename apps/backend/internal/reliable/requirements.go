package reliable

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var errRequirementsChanged = errors.New("requirements_changed: new user requirements need a fresh audit")
var errNewBatch = errors.New("new_batch_required: start a linked modification task")

func (e *Engine) addRequirement(task *Task, message *TaskMessage) error {
	if acceptancePending(task) {
		return errors.New("local_checkpoint_pending: finish saving the accepted result before changing requirements")
	}
	if task.SeedPending != nil {
		return errors.New("renewal_active: wait for inherited source validation before adding requirements")
	}
	if task.RecoveryPending != nil {
		return errors.New("recovery_active: wait for retained candidate validation before adding requirements")
	}
	if task.ReadOnly || task.State == stateAccepted {
		return errNewBatch
	}
	if task.State == stateCancelled || task.State == stateCancelling || task.State == stateApplying {
		return errors.New("requirement_unavailable: this task cannot receive modification requirements")
	}
	if task.Round >= MaxRounds {
		return errors.New("batch_exhausted: this work batch has used its three automatic attempts")
	}
	if err := e.promoteFuturePlan(task); err != nil {
		return err
	}
	task.RequirementRevision++
	message.Revision = task.RequirementRevision
	name := fmt.Sprintf("requirement-%d-%s.json", message.Revision, message.ID)
	if err := e.saveJSON(task.ID, name, message); err != nil {
		return err
	}
	task.RequirementEvidence = append(task.RequirementEvidence, name)
	task.EvidenceHash = ""
	if conversationIdle(task.State) {
		task.State, task.ContinuePending = stateQueued, true
		task.Reason = "New user requirements will be audited before another implementation attempt"
	}
	return nil
}

func (e *Engine) freezeRequirements(task *Task) error {
	current, err := e.Store.Update(task.ID, func(saved *Task) error {
		if saved.State == stateCancelling || saved.State == stateCancelled {
			return context.Canceled
		}
		saved.ContinuePending = false
		return nil
	})
	if err != nil {
		return err
	}
	current.AuditedRevision = current.RequirementRevision
	*task = *current
	return nil
}

func (e *Engine) commitAudit(ctx context.Context, task *Task, audit *Audit) error {
	frozenRevision := task.AuditedRevision
	current, err := e.Store.Update(task.ID, func(saved *Task) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if saved.State == stateCancelling || saved.State == stateCancelled {
			return context.Canceled
		}
		if saved.Audit != nil {
			saved.ApprovedPaths = unionPaths(saved.ApprovedPaths, saved.Audit.AllowedPaths)
		}
		saved.ApprovedPaths = unionPaths(saved.ApprovedPaths, audit.AllowedPaths)
		saved.Audit, saved.AuditedRevision = audit, frozenRevision
		saved.ContinuePending = saved.RequirementRevision != frozenRevision
		for i := range saved.Messages {
			message := &saved.Messages[i]
			if message.Kind == messageRequirement && message.Status == stateQueued && message.Revision <= frozenRevision {
				message.Status, message.AppliedRound = "applied", saved.Round+1
			}
		}
		return nil
	})
	if err == nil {
		*task = *current
	}
	return err
}

func taskGoal(task *Task) string {
	goal := task.Goal
	for _, message := range task.Messages {
		if message.Kind == messageRequirement && message.Revision <= task.AuditedRevision {
			goal += fmt.Sprintf("\nUSER REQUIREMENT %d:\n%s", message.Revision, message.Content)
		}
	}
	return strings.TrimSpace(goal)
}
