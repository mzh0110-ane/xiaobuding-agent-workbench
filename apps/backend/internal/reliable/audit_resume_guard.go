package reliable

import (
	"errors"
	"fmt"
)

func (e *Engine) validateAuditResumeSource(task *Task, current *Snapshot) error {
	approved := false
	if auditReplanPending(task) {
		var err error
		approved, err = e.auditReplanApproved(task, current)
		if err != nil {
			return err
		}
		if !approved {
			return errors.New("audit_resume_source_changed: source changed or planning settings invalidated the human confirmation; inspect and confirm again")
		}
	}
	plannedRound := task.Round + 1
	for i := len(task.Runs) - 1; i >= 0; i-- {
		run := task.Runs[i]
		if run.Round != plannedRound {
			continue
		}
		if run.Role == roleImplementer {
			return nil
		}
		if run.Role != roleAuditor || run.Status == dispatchNotStarted {
			continue
		}
		if run.SourceHash == "" {
			return errors.New("audit_resume_evidence: previous planning dispatch has no source identity")
		}
		if current.Hash != run.SourceHash {
			if approved {
				return nil
			}
			return fmt.Errorf("audit_resume_source_changed: read-only planning source changed after dispatch %s; retained changes require explicit review before replanning", run.ID)
		}
		return nil
	}
	return nil
}
