package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
)

const maxStageAttempts = 3

type domainInvocationError struct{ error }

func (e *domainInvocationError) Unwrap() error { return e.error }

type StageRecovery struct {
	Round               int    `json:"round"`
	Domain              string `json:"domain"`
	Attempts            int    `json:"attempts"`
	ManualResumes       int    `json:"manual_resumes"`
	Status              string `json:"status"`
	SourceHash          string `json:"source_hash"`
	RequirementRevision int    `json:"requirement_revision"`
	AuditHash           string `json:"audit_hash"`
	LastError           string `json:"last_error,omitempty"`
}

func domainStageResumeEligible(t *Task) bool {
	if auditReplanPending(t) {
		return false
	}
	if !domainStageInputsReady(t) || validateStageIdentity(t) != nil {
		return false
	}
	if stage := t.StageRecovery; stage != nil && stage.Round == t.Round && stage.Status != "completed" {
		return true
	}
	if len(t.Runs) == 0 || laterAuditAttempted(t) {
		return false
	}
	last := t.Runs[len(t.Runs)-1]
	return failedDomainDispatch(last, t.Round)
}

func laterAuditAttempted(t *Task) bool {
	for _, run := range t.Runs {
		if run.Role == roleAuditor && run.Round > t.Round && run.Status != dispatchNotStarted {
			return true
		}
	}
	return false
}

func domainStageInputsReady(t *Task) bool {
	if t == nil {
		return false
	}
	return t.ModelPlan.Mode != "" && t.Round > 0 && t.Audit != nil && t.Worktree != "" && t.RequirementRevision == t.AuditedRevision && t.RecoveryPending == nil && len(t.RecoveryEvidence) == 0 && t.SeedPending == nil
}

func failedDomainDispatch(run Dispatch, round int) bool {
	return run.Role == roleImplementer && run.Round == round && (run.Status == "failed" || run.Status == "interrupted") && run.Domain != ""
}

func operationalStageError(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"context deadline exceeded", "timed out", "timeout", "connection reset", "connection refused", "temporary failure", "rate limit", "too many requests", "http 429", "http 502", "http 503", "http 504"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func (e *Engine) markDomainStage(t *Task, domain, status, source string, lastErr error, manual bool) error {
	audit, _ := json.Marshal(t.Audit)
	updated, err := e.Store.Update(t.ID, func(saved *Task) error {
		stage := saved.StageRecovery
		if stage == nil || stage.Round != t.Round || stage.Domain != domain {
			stage = &StageRecovery{Round: t.Round, Domain: domain, Attempts: stageDispatchAttempts(saved, domain), RequirementRevision: t.AuditedRevision, AuditHash: digest(audit)}
		}
		if status == "running" {
			stage.Attempts++
		}
		if manual {
			stage.ManualResumes++
		}
		stage.Status, stage.SourceHash = status, source
		stage.LastError = ""
		if lastErr != nil {
			stage.LastError = lastErr.Error()
		}
		saved.StageRecovery = stage
		return nil
	})
	if err == nil {
		t.StageRecovery = updated.StageRecovery
	}
	return err
}

func (e *Engine) resumeDomainStage(ctx context.Context, t *Task) error {
	if err := e.processQuiescent(t); err != nil {
		return err
	}
	if err := e.restoreWorktreeContract(ctx, t); err != nil {
		return err
	}
	if err := e.setState(t.ID, stateImplementing, t.Reason); err != nil {
		return err
	}
	emitActivity(ctx, "phase", "继续未完成的执行阶段", "核对已完成阶段的证据与源码；运行重试不增加代码返工轮次。")
	if err := e.implementDomains(ctx, t); err != nil {
		return err
	}
	decision, reason, err := e.verifyAndReview(ctx, t)
	if errors.Is(err, errRequirementsChanged) {
		decision, reason, err = Rework, "New user requirements require a fresh audit and verification", nil
	}
	if err != nil {
		return err
	}
	if decision == Pass {
		return nil
	}
	if decision == Block {
		return errors.New(reason)
	}
	if t.Round >= MaxRounds {
		_ = e.setState(t.ID, stateExhausted, reason)
		return fmt.Errorf("three rounds exhausted: %s", reason)
	}
	if err = e.setState(t.ID, stateAuditing, reason); err != nil {
		return err
	}
	t, err = e.Store.Get(t.ID)
	if err != nil {
		return err
	}
	return e.runRounds(ctx, t)
}

func (e *Engine) implementDomainWithRetry(ctx context.Context, t *Task, domain string, previous []domainOutcome) (*domainOutcome, error) {
	manual := stageHasPriorAttempt(t, domain)
	for attempt := 0; attempt < maxStageAttempts; attempt++ {
		before, err := TakeSnapshot(ctx, t.Worktree)
		if err != nil {
			return nil, err
		}
		if err = e.markDomainStage(t, domain, "running", before.Hash, nil, manual && attempt == 0); err != nil {
			return nil, err
		}
		outcome, runErr := e.implementDomain(ctx, t, domain, previous, before)
		if runErr == nil {
			return outcome, e.markDomainStage(t, domain, "completed", outcome.After.Hash, nil, false)
		}
		retry, runErr := e.recordDomainFailure(ctx, t, domain, before, runErr)
		if !retry || attempt == maxStageAttempts-1 || ctx.Err() != nil {
			return nil, runErr
		}
		emitActivity(ctx, "phase", "执行连接中断，重试当前阶段", fmt.Sprintf("%s · 运行尝试 %d/%d；保留之前的成果", domain, attempt+2, maxStageAttempts))
	}
	return nil, errors.New("stage_retry_limit: automatic operational retry limit reached")
}

func (e *Engine) recordDomainFailure(ctx context.Context, t *Task, domain string, before *Snapshot, runErr error) (bool, error) {
	after, snapErr := TakeSnapshot(ctx, t.Worktree)
	if snapErr == nil {
		snapErr = validateFailedDomainSource(t, before, after, domain)
	}
	if snapErr != nil {
		runErr = fmt.Errorf("stage_resume_source_changed: %w", snapErr)
	}
	status := stateBlocked
	var invocation *domainInvocationError
	if snapErr == nil && errors.As(runErr, &invocation) && operationalStageError(runErr) {
		status = "retryable"
	}
	if err := e.markDomainStage(t, domain, status, before.Hash, runErr, false); err != nil {
		return false, err
	}
	return status == "retryable", runErr
}

func stageHasPriorAttempt(t *Task, domain string) bool {
	if stageDispatchAttempts(t, domain) > 0 {
		return true
	}
	stage := t.StageRecovery
	return stage != nil && stage.Round == t.Round && stage.Domain == domain && stage.Attempts > 0
}

func stageDispatchAttempts(t *Task, domain string) int {
	count := 0
	for _, run := range t.Runs {
		if run.Role == roleImplementer && run.Round == t.Round && run.Domain == domain {
			count++
		}
	}
	return count
}

func validateFailedDomainSource(t *Task, before, after *Snapshot, domain string) error {
	if before == nil || after == nil || t.Baseline == nil || after.Head != t.Base || after.ResourceHash != t.Baseline.ResourceHash {
		return errors.New("source baseline or Git history changed")
	}
	if err := checkScope(changedFiles(t.Baseline, after), t.ApprovedPaths); err != nil {
		return err
	}
	return checkScope(changedFiles(before, after), auditDomainPaths(t.Audit, domain))
}
