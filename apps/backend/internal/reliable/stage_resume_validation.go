package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
)

func (e *Engine) readStageEvidence(t *Task, name string, value any) error {
	if !safeRelative(name) {
		return errors.New("stage_resume_evidence: unsafe artifact path")
	}
	root, err := os.OpenRoot(filepath.Join(e.Root, "evidence", t.ID))
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	f, err := root.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 16<<20 {
		return errors.New("stage_resume_evidence: invalid artifact")
	}
	raw, err := io.ReadAll(io.LimitReader(f, (16<<20)+1))
	if err != nil {
		return err
	}
	return decodeStrict(raw, value)
}

func (e *Engine) prepareDomainOutcomes(ctx context.Context, t *Task) ([]domainOutcome, error) {
	if err := validateStageIdentity(t); err != nil {
		return nil, err
	}
	reference, created, err := e.loadDomainRound(ctx, t)
	if err != nil || created {
		return nil, err
	}
	outcomes, reference, err := e.loadCompletedDomains(t, reference)
	if err != nil {
		return nil, err
	}
	current, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return nil, err
	}
	domains := t.ModelPlan.domains()
	if len(outcomes) == len(domains) {
		if current.Hash != reference.Hash {
			return nil, errors.New("stage_resume_source_changed: completed source changed")
		}
		return outcomes, nil
	}
	if err = e.validateResumeEntry(t, domains[len(outcomes)], reference, current); err != nil {
		return nil, err
	}
	if err = validateFailedDomainSource(t, reference, current, domains[len(outcomes)]); err != nil {
		return nil, fmt.Errorf("stage_resume_source_changed: %w", err)
	}
	return outcomes, nil
}

func (e *Engine) loadDomainRound(ctx context.Context, t *Task) (*Snapshot, bool, error) {
	auditName := fmt.Sprintf("round-%d-audit.json", t.Round)
	beforeName := fmt.Sprintf("round-%d-before.json", t.Round)
	var savedAudit Audit
	if err := e.readStageEvidence(t, auditName, &savedAudit); os.IsNotExist(err) {
		if err = e.saveJSON(t.ID, auditName, t.Audit); err != nil {
			return nil, false, err
		}
		before, snapErr := TakeSnapshot(ctx, t.Worktree)
		if snapErr != nil {
			return nil, false, snapErr
		}
		return before, true, e.saveJSON(t.ID, beforeName, before)
	} else if err != nil {
		return nil, false, err
	}
	if !reflect.DeepEqual(&savedAudit, t.Audit) || t.RequirementRevision != t.AuditedRevision {
		return nil, false, errors.New("stage_resume_invalidated: requirements or the frozen audit changed")
	}
	var reference Snapshot
	if err := e.readStageEvidence(t, beforeName, &reference); err != nil {
		return nil, false, err
	}
	if reference.Head != t.Base || t.Baseline == nil || reference.ResourceHash != t.Baseline.ResourceHash {
		return nil, false, errors.New("stage_resume_invalidated: initial source identity changed")
	}
	return &reference, false, nil
}

func (e *Engine) loadCompletedDomains(t *Task, reference *Snapshot) ([]domainOutcome, *Snapshot, error) {
	var outcomes []domainOutcome
	for _, domain := range t.ModelPlan.domains() {
		name := fmt.Sprintf("round-%d-%s-implementation.json", t.Round, domain)
		if !containsPath(t.DomainEvidence, name) {
			break
		}
		var outcome domainOutcome
		if err := e.readStageEvidence(t, name, &outcome); err != nil {
			return nil, nil, err
		}
		if err := e.validateSavedDomain(t, domain, reference, &outcome); err != nil {
			return nil, nil, err
		}
		outcomes = append(outcomes, outcome)
		reference = outcome.After
	}
	return outcomes, reference, nil
}

func validateStageIdentity(t *Task) error {
	stage := t.StageRecovery
	if stage == nil || stage.Round != t.Round {
		return nil
	}
	raw, _ := json.Marshal(t.Audit)
	if stage.AuditHash != digest(raw) || stage.RequirementRevision != t.AuditedRevision {
		return errors.New("stage_resume_invalidated: frozen stage inputs changed")
	}
	return nil
}

func (e *Engine) validateResumeEntry(t *Task, domain string, reference, current *Snapshot) error {
	for _, run := range t.Runs {
		if run.Role != roleImplementer || run.Round != t.Round || run.Domain != domain {
			continue
		}
		var input handoffInput
		if err := e.readStageEvidence(t, run.InputEvidence, &input); err != nil {
			return err
		}
		if run.SourceHash != reference.Hash || input.SourceHash != reference.Hash || input.TaskID != t.ID || input.Base != t.Base || input.RequirementRevision != t.AuditedRevision || !reflect.DeepEqual(input.Audit, t.Audit) {
			return errors.New("stage_resume_evidence: interrupted writer handoff does not follow completed source")
		}
		return nil
	}
	if reference.Hash != current.Hash {
		return errors.New("stage_resume_source_changed: source changed before the next writer started")
	}
	return nil
}

func (e *Engine) validateSavedDomain(t *Task, domain string, reference *Snapshot, outcome *domainOutcome) error {
	if outcome.Domain != domain || outcome.Before == nil || outcome.After == nil {
		return errors.New("stage_resume_evidence: completed writer source chain changed")
	}
	if !reflect.DeepEqual(outcome.Before, reference) {
		if err := validateFailedDomainSource(t, reference, outcome.Before, domain); err != nil {
			return err
		}
		if err := e.validateResumeEntry(t, domain, reference, outcome.Before); err != nil {
			return err
		}
	}
	if err := validateFailedDomainSource(t, outcome.Before, outcome.After, domain); err != nil {
		return err
	}
	if err := checkReads(outcome.Report.ReadFiles, outcome.Before, outcome.After); err != nil {
		return err
	}
	if err := validateImplementation(outcome.Report, t.Audit.RequiredReadFiles); err != nil {
		return err
	}
	completed := completedDomainDispatch(t, domain)
	if completed == nil || completed.Session == "" || completed.SourceHash != outcome.Before.Hash {
		return errors.New("stage_resume_evidence: completed writer dispatch missing")
	}
	return e.validateSavedDomainDispatch(t, completed, outcome)
}

func completedDomainDispatch(t *Task, domain string) *Dispatch {
	var completed *Dispatch
	for i := range t.Runs {
		run := &t.Runs[i]
		if run.Role == roleImplementer && run.Domain == domain && run.Round == t.Round && run.Status == "completed" {
			completed = run
		}
	}
	return completed
}

func (e *Engine) validateSavedDomainDispatch(t *Task, completed *Dispatch, outcome *domainOutcome) error {
	var input handoffInput
	if err := e.readStageEvidence(t, completed.InputEvidence, &input); err != nil {
		return err
	}
	if input.TaskID != t.ID || input.Base != t.Base || input.SourceHash != outcome.Before.Hash || input.RequirementRevision != t.AuditedRevision || !reflect.DeepEqual(input.Audit, t.Audit) || input.Dispatch.ID != completed.ID {
		return errors.New("stage_resume_evidence: completed writer handoff changed")
	}
	var result struct {
		Dispatch Dispatch        `json:"dispatch"`
		Report   json.RawMessage `json:"report"`
	}
	if err := e.readStageEvidence(t, completed.EndEvidence, &result); err != nil {
		return err
	}
	if !reflect.DeepEqual(result.Dispatch, *completed) {
		return errors.New("stage_resume_evidence: completed writer result identity changed")
	}
	paths, clean, err := resourceRequestOutput(result.Report)
	if err != nil {
		return err
	}
	var report implementation
	if len(paths) != 0 || decodeStrict(clean, &report) != nil || !reflect.DeepEqual(report, outcome.Report) {
		return errors.New("stage_resume_evidence: completed writer report changed")
	}
	return nil
}

func (e *Engine) saveDomainOutcomes(t *Task, outcomes []domainOutcome) error {
	name := fmt.Sprintf("round-%d-implementation.json", t.Round)
	var saved []domainOutcome
	if err := e.readStageEvidence(t, name, &saved); err == nil {
		if !reflect.DeepEqual(saved, outcomes) {
			return errors.New("stage_resume_evidence: final writer outcomes changed")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return e.saveJSON(t.ID, name, outcomes)
}
