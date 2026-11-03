package reliable

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type AuditReplanApproval struct {
	Receipt             string `json:"receipt"`
	ReceiptHash         string `json:"receipt_hash"`
	SourceHash          string `json:"source_hash"`
	Round               int    `json:"round"`
	RequirementRevision int    `json:"requirement_revision"`
	PlanRevision        int    `json:"plan_revision"`
}

type AuditReplanInspection struct {
	Eligible            bool     `json:"eligible"`
	SourceHash          string   `json:"source_hash"`
	ChangedFiles        []string `json:"changed_files"`
	TotalChanges        int      `json:"total_changes"`
	AllowedPaths        []string `json:"allowed_paths"`
	Reason              string   `json:"reason"`
	Round               int      `json:"round"`
	RequirementRevision int      `json:"requirement_revision"`
	PlanRevision        int      `json:"plan_revision"`
}

func auditReplanPending(task *Task) bool {
	if task == nil || task.AuditReplan == nil {
		return false
	}
	return task.AuditReplan.Round == task.Round
}

func auditReplanState(task *Task) error {
	if task.ReadOnly || task.ArchivedAt != "" || task.SuccessorTaskID != "" || task.Round >= MaxRounds || task.SeedPending != nil || task.RecoveryPending != nil || acceptancePending(task) {
		return errors.New("audit_replan_ineligible: this task cannot start a fresh plan within its current batch")
	}
	if task.State != stateBlocked && task.State != stateInterrupted {
		return errors.New("audit_replan_ineligible: stop the current execution before confirming a fresh plan")
	}
	return nil
}

func (e *Engine) auditReplanCandidate(ctx context.Context, id string) (*Task, *Snapshot, error) {
	task, err := e.Store.Get(id)
	if err != nil {
		return nil, nil, err
	}
	if err = auditReplanState(task); err != nil {
		return task, nil, err
	}
	candidate, snapshot, err := e.resultCandidate(ctx, id)
	if err != nil {
		return task, snapshot, err
	}
	task = candidate
	if err = auditReplanState(task); err != nil {
		return task, snapshot, err
	}
	var contract Task
	if err = e.readRecoveryJSON(task.ID, "contract.json", &contract); err != nil || !sameRecoveryContract(&contract, task) {
		return task, snapshot, errors.New("audit_replan_evidence: original project contract is missing or changed")
	}
	if err = recoveryOriginal(ctx, task); err != nil {
		return task, snapshot, fmt.Errorf("audit_replan_changed: original project changed: %w", err)
	}
	return task, snapshot, auditReplanSource(task, snapshot)
}

func auditReplanSource(task *Task, snapshot *Snapshot) error {
	if snapshot.Head != task.Base || snapshot.ResourceHash != task.Baseline.ResourceHash {
		return errors.New("audit_replan_changed: candidate history or frozen inputs changed")
	}
	if len(task.AllowedPaths) == 0 {
		return errors.New("audit_replan_scope: original authorization scope is missing")
	}
	changed := changedFiles(task.Baseline, snapshot)
	for _, name := range changed {
		if protectedPath(strings.ToLower(name)) {
			return fmt.Errorf("audit_replan_scope: protected file changed: %s", name)
		}
	}
	if err := checkScope(changed, task.AllowedPaths); err != nil {
		return fmt.Errorf("audit_replan_scope: %w", err)
	}
	return nil
}

func (e *Engine) InspectAuditReplan(ctx context.Context, id string) (*AuditReplanInspection, error) {
	task, snapshot, err := e.auditReplanCandidate(ctx, id)
	if task == nil {
		return nil, err
	}
	report := &AuditReplanInspection{ChangedFiles: []string{}, AllowedPaths: append([]string{}, task.AllowedPaths...), Round: task.Round, RequirementRevision: task.RequirementRevision, PlanRevision: task.PlanRevision}
	if snapshot != nil {
		report.SourceHash, report.ChangedFiles = snapshot.Hash, changedFiles(task.Baseline, snapshot)
		report.TotalChanges = len(report.ChangedFiles)
	}
	if err != nil {
		report.Reason = err.Error()
		return report, nil
	}
	report.Eligible, report.Reason = true, "audit_replan_ready"
	return report, nil
}
