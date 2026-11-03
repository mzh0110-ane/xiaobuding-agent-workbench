package reliable

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type AuditReplanResult struct {
	Queued              bool   `json:"queued"`
	AlreadyConfirmed    bool   `json:"already_confirmed"`
	SourceHash          string `json:"source_hash"`
	Receipt             string `json:"receipt"`
	Round               int    `json:"round"`
	RequirementRevision int    `json:"requirement_revision"`
	PlanRevision        int    `json:"plan_revision"`
}

type auditReplanExecution struct {
	context context.Context
	release func()
	result  *AuditReplanResult
}

func (e *Engine) ConfirmAuditReplan(ctx context.Context, id string, request auditReplanRequest) (*AuditReplanResult, error) {
	request.Note = strings.TrimSpace(request.Note)
	if _, err := hex.DecodeString(request.SourceHash); err != nil || len(request.SourceHash) != 64 || !request.Confirm || !validOptionalProjectText(request.Note, 2000) {
		return nil, errors.New("audit_replan_confirmation: explicitly confirm the inspected source fingerprint")
	}
	task, snapshot, err := e.auditReplanCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	if snapshot.Hash != request.SourceHash {
		return nil, errors.New("audit_replan_changed: candidate changed after inspection; inspect it again")
	}
	e.mu.Lock()
	execution, err := e.confirmAuditReplanLocked(ctx, task, request)
	e.mu.Unlock()
	if err != nil {
		if execution != nil && execution.release != nil {
			execution.release()
		}
		return nil, err
	}
	if execution.release != nil {
		go func() {
			defer execution.release()
			_ = e.runTask(execution.context, id)
		}()
	}
	return execution.result, nil
}

func (e *Engine) confirmAuditReplanLocked(ctx context.Context, task *Task, request auditReplanRequest) (*auditReplanExecution, error) {
	if e.closing || e.active[task.ID] != nil {
		return nil, errors.New("audit_replan_ineligible: this task is already executing or shutting down")
	}
	fresh, err := e.Store.Get(task.ID)
	if err != nil {
		return nil, err
	}
	if fresh.Updated != task.Updated || auditReplanState(fresh) != nil {
		return nil, errors.New("audit_replan_changed: task settings or lifecycle changed after inspection")
	}
	snapshot, err := TakeSnapshot(ctx, fresh.Worktree)
	if err != nil {
		return nil, err
	}
	if snapshot.Hash != request.SourceHash {
		return nil, errors.New("audit_replan_changed: candidate changed before confirmation")
	}
	if err = auditReplanSource(fresh, snapshot); err != nil {
		return nil, err
	}
	approved, err := e.auditReplanApproved(fresh, snapshot)
	if err != nil {
		return nil, err
	}
	if approved {
		return &auditReplanExecution{result: auditReplanResponse(fresh.AuditReplan, false)}, nil
	}
	runContext, release, err := e.reserveLocked(context.Background(), fresh.ID)
	if err != nil {
		return nil, err
	}
	execution := &auditReplanExecution{context: runContext, release: release}
	approval, err := e.saveAuditReplanApproval(ctx, fresh, request, changedFiles(fresh.Baseline, snapshot))
	if err == nil {
		execution.result = auditReplanResponse(approval, true)
	}
	return execution, err
}

func auditReplanResponse(approval *AuditReplanApproval, queued bool) *AuditReplanResult {
	return &AuditReplanResult{Queued: queued, AlreadyConfirmed: !queued, SourceHash: approval.SourceHash, Receipt: approval.Receipt, Round: approval.Round, RequirementRevision: approval.RequirementRevision, PlanRevision: approval.PlanRevision}
}

func (e *Engine) saveAuditReplanApproval(ctx context.Context, task *Task, request auditReplanRequest, confirmed ...[]string) (*AuditReplanApproval, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var paths []string
	if len(confirmed) > 0 {
		paths = append([]string{}, confirmed[0]...)
	}
	receipt := auditReplanReceipt{Version: auditReplanReceiptVersion, TaskID: task.ID, Project: task.Project, Worktree: task.Worktree, Base: task.Base,
		SourceHash: request.SourceHash, Round: task.Round, RequirementRevision: task.RequirementRevision, PlanRevision: task.PlanRevision,
		AllowedPaths: append([]string{}, task.AllowedPaths...), ConfirmedPaths: paths, Note: request.Note, PreviousReason: projectExcerpt(task.Reason, 1000), ApprovedAt: time.Now().UTC().Format(time.RFC3339Nano), PreviousApproval: task.AuditReplan}
	name := "audit-replan-" + randomID() + ".json"
	if err := e.saveJSON(task.ID, name, receipt); err != nil {
		return nil, err
	}
	raw, err := e.readAuditReplanReceipt(task.ID, name)
	if err != nil {
		return nil, err
	}
	approval := &AuditReplanApproval{Receipt: name, ReceiptHash: digest(raw), SourceHash: request.SourceHash, Round: task.Round, RequirementRevision: task.RequirementRevision, PlanRevision: task.PlanRevision}
	_, err = e.Store.Update(task.ID, func(saved *Task) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if saved.Updated != task.Updated || auditReplanState(saved) != nil {
			return errors.New("audit_replan_changed: task changed while saving human confirmation")
		}
		saved.AuditReplan = approval
		saved.RequirementEvidence = append(saved.RequirementEvidence, name)
		saved.ApprovedPaths = unionPaths(saved.ApprovedPaths, paths)
		saved.State, saved.ContinuePending = stateQueued, false
		saved.Reason = "Human confirmed the current candidate within the original authorization scope; a fresh audit is required."
		if request.Note != "" {
			saved.Reason += " Human note: " + request.Note
		}
		return nil
	})
	return approval, err
}
