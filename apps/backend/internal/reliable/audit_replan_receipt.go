package reliable

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const auditReplanReceiptVersion = "human-audit-replan-v1"

type auditReplanReceipt struct {
	Version             string               `json:"version"`
	TaskID              string               `json:"task_id"`
	Project             string               `json:"project"`
	Worktree            string               `json:"worktree"`
	Base                string               `json:"base"`
	SourceHash          string               `json:"source_hash"`
	Round               int                  `json:"round"`
	RequirementRevision int                  `json:"requirement_revision"`
	PlanRevision        int                  `json:"plan_revision"`
	AllowedPaths        []string             `json:"allowed_paths"`
	ConfirmedPaths      []string             `json:"confirmed_paths,omitempty"`
	ApprovedAt          string               `json:"approved_at"`
	PreviousReason      string               `json:"previous_reason"`
	Note                string               `json:"note"`
	PreviousApproval    *AuditReplanApproval `json:"previous_approval,omitempty"`
}

func (e *Engine) readAuditReplanReceipt(id, name string) ([]byte, error) {
	if !safeRelative(name) || strings.ContainsAny(name, "/\\") || !strings.HasPrefix(name, "audit-replan-") || !strings.HasSuffix(name, ".json") {
		return nil, errors.New("audit_replan_evidence: invalid human confirmation receipt")
	}
	root, err := os.OpenRoot(filepath.Join(e.Root, "evidence", id))
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16<<20 {
		return nil, errors.New("audit_replan_evidence: confirmation receipt is unavailable or oversized")
	}
	raw, err := io.ReadAll(io.LimitReader(file, (16<<20)+1))
	if err != nil || len(raw) > 16<<20 {
		return nil, errors.New("audit_replan_evidence: confirmation receipt could not be read")
	}
	return raw, nil
}

func (e *Engine) auditReplanApproved(task *Task, current *Snapshot) (bool, error) {
	if !auditReplanPending(task) || task.AuditReplan.SourceHash != current.Hash || task.AuditReplan.RequirementRevision != task.RequirementRevision || task.AuditReplan.PlanRevision != task.PlanRevision {
		return false, nil
	}
	if err := e.validateAuditReplanEvidence(task, nil); err != nil {
		return false, err
	}
	return true, nil
}

func auditReplanReceiptMatches(receipt *auditReplanReceipt, task *Task, approval *AuditReplanApproval) bool {
	return receipt.Version == auditReplanReceiptVersion && receipt.TaskID == task.ID && receipt.Project == task.Project &&
		receipt.Worktree == task.Worktree && receipt.Base == task.Base && receipt.SourceHash == approval.SourceHash &&
		receipt.Round == approval.Round && receipt.RequirementRevision == approval.RequirementRevision && receipt.PlanRevision == approval.PlanRevision &&
		reflect.DeepEqual(receipt.AllowedPaths, task.AllowedPaths)
}

// Approval authentication is historical. Later source, requirement and model
// revisions do not authorize changing the bytes of an earlier human receipt.
func (e *Engine) validateAuditReplanEvidence(task *Task, hashes map[string]string) error {
	seen := make(map[string]bool)
	for approval := task.AuditReplan; approval != nil; {
		if len(seen) >= 200 || seen[approval.Receipt] || !containsPath(task.RequirementEvidence, approval.Receipt) {
			return errors.New("audit_replan_evidence: human approval history is missing, cyclic or oversized")
		}
		raw, err := e.readAuditReplanReceipt(task.ID, approval.Receipt)
		if err != nil || digest(raw) != approval.ReceiptHash {
			return errors.New("audit_replan_evidence: human confirmation receipt is missing or changed")
		}
		if hashes != nil && hashes[approval.Receipt] != approval.ReceiptHash {
			return errors.New("audit_replan_evidence: sealed receipt bytes differ from the human confirmation")
		}
		var receipt auditReplanReceipt
		if err = decodeStrict(raw, &receipt); err != nil || !auditReplanReceiptMatches(&receipt, task, approval) {
			return errors.New("audit_replan_evidence: human confirmation does not match its immutable approval")
		}
		seen[approval.Receipt] = true
		approval = receipt.PreviousApproval
	}
	for _, name := range task.RequirementEvidence {
		if strings.HasPrefix(name, "audit-replan-") && !seen[name] {
			return errors.New("audit_replan_evidence: a historical human receipt has no authenticated approval")
		}
	}
	return nil
}
