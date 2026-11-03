package reliable

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const auditRuntimeWindow = 30 * time.Minute

func (e *Engine) callAuditWithRetry(ctx context.Context, task *Task, prompt string, before *Snapshot) (*AgentResult, error) {
	window, cancel := context.WithTimeout(ctx, auditRuntimeWindow)
	defer cancel()
	identity, err := e.Store.Get(task.ID)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < maxStageAttempts; attempt++ {
		result, runErr := e.call(window, task, roleAuditor, prompt, task.Round+1)
		if runErr == nil {
			return result, nil
		}
		if window.Err() != nil || !operationalStageError(runErr) || attempt == maxStageAttempts-1 {
			return nil, runErr
		}
		if err = e.validateAuditRetry(window, identity, before); err != nil {
			return nil, err
		}
		emitActivity(window, "phase", "规划连接中断，重试只读调查", fmt.Sprintf("运行尝试 %d/%d；共享本次 30 分钟规划时限，不增加代码返工轮次。", attempt+2, maxStageAttempts))
	}
	return nil, errors.New("audit_retry_limit: planning runtime retry limit reached")
}

func (e *Engine) validateAuditRetry(ctx context.Context, beforeTask *Task, before *Snapshot) error {
	proof, _ := ctx.Value(ownedCleanupKey{}).(*ownedCleanupProof)
	if !proof.verified() {
		return errors.New("audit_retry_invalidated: prior owned process cleanup was not confirmed")
	}
	afterTask, err := e.Store.Get(beforeTask.ID)
	if err != nil {
		return err
	}
	if beforeTask.RequirementRevision != afterTask.RequirementRevision || beforeTask.AuditedRevision != afterTask.AuditedRevision || beforeTask.PlanRevision != afterTask.PlanRevision || afterTask.State != stateAuditing {
		return errors.New("audit_retry_invalidated: frozen requirements or model selection changed")
	}
	if len(afterTask.Runs) == 0 {
		return errors.New("audit_retry_invalidated: failed invocation has no dispatch evidence")
	}
	last := afterTask.Runs[len(afterTask.Runs)-1]
	if last.Role != roleAuditor || last.Status != "failed" || last.SourceHash != before.Hash {
		return errors.New("audit_retry_invalidated: only a failed auditor invocation may be retried")
	}
	after, err := TakeSnapshot(ctx, beforeTask.Worktree)
	if err != nil {
		return err
	}
	if before.Hash != after.Hash || before.Head != after.Head || before.ResourceHash != after.ResourceHash {
		return errors.New("audit_retry_invalidated: read-only planning source changed")
	}
	return nil
}
