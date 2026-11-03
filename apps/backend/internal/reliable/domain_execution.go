package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
)

type domainOutcome struct {
	Domain string         `json:"domain"`
	Before *Snapshot      `json:"before"`
	After  *Snapshot      `json:"after"`
	Report implementation `json:"report"`
}

func (e *Engine) implementDomains(ctx context.Context, t *Task) error {
	outcomes, err := e.prepareDomainOutcomes(ctx, t)
	if err != nil {
		return err
	}
	for _, domain := range t.ModelPlan.domains()[len(outcomes):] {
		if err = ctx.Err(); err != nil {
			return err
		}
		outcome, domainErr := e.implementDomainWithRetry(ctx, t, domain, outcomes)
		if domainErr != nil {
			return domainErr
		}
		outcomes = append(outcomes, *outcome)
	}
	return e.saveDomainOutcomes(t, outcomes)
}

func (e *Engine) implementDomain(ctx context.Context, t *Task, domain string, previous []domainOutcome, before *Snapshot) (*domainOutcome, error) {
	data, _ := json.Marshal(map[string]any{"goal": taskGoal(t), "domain": domain, "allowed_paths": auditDomainPaths(t.Audit, domain), "audit": t.Audit, "prior_domain_outcomes": domainHandoffs(t, previous), "round": t.Round, "max_rounds": MaxRounds, "previous_failure": t.Reason, "previous_check_directory": filepath.Join(e.Root, "evidence", t.ID, fmt.Sprintf("round-%d", t.Round-1))})
	prompt := "\nROLE: IMPLEMENTER. You are the sole CURRENT writer, responsible ONLY for this domain's allowed_paths. Freshly read every audit.required_read_files path and acknowledge them in read_files. Read current source and the preceding writer's changes before editing; preserve their work. Implement this domain's share of the goal, respecting the audit's API contract. Do not edit another domain, shared files assigned to another writer, tests to weaken them, or files outside the worktree. The next domain runs afterward, then the controller verifies the complete result. Inputs:\n" + string(data)
	prompt += automaticCheckInstructions(t)
	result, err := e.call(ctx, t, roleImplementer, prompt, t.Round, domain)
	if err != nil {
		return nil, &domainInvocationError{err}
	}
	after, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return nil, err
	}
	if after.Head != t.Base || t.Baseline == nil {
		return nil, errors.New("domain writer changed Git history or lost the source baseline")
	}
	if err = checkScope(changedFiles(t.Baseline, after), t.ApprovedPaths); err != nil {
		return nil, err
	}
	if err = checkScope(changedFiles(before, after), auditDomainPaths(t.Audit, domain)); err != nil {
		return nil, fmt.Errorf("%s writer exceeded its domain: %w", domain, err)
	}
	var report implementation
	if err = decodeStrict(result.Output, &report); err != nil {
		return nil, err
	}
	if err = checkReads(report.ReadFiles, before, after); err != nil {
		return nil, err
	}
	if err = validateImplementation(report, t.Audit.RequiredReadFiles); err != nil {
		return nil, err
	}
	outcome := &domainOutcome{Domain: domain, Before: before, After: after, Report: report}
	name := fmt.Sprintf("round-%d-%s-implementation.json", t.Round, domain)
	_, err = e.Store.Update(t.ID, func(saved *Task) error {
		if saveErr := e.saveJSON(t.ID, name, outcome); saveErr != nil {
			return saveErr
		}
		saved.DomainEvidence = append(saved.DomainEvidence, name)
		return nil
	})
	return outcome, err
}

func domainHandoffs(t *Task, outcomes []domainOutcome) []map[string]any {
	briefs := make([]map[string]any, 0, len(outcomes))
	for _, outcome := range outcomes {
		briefs = append(briefs, map[string]any{"domain": outcome.Domain, "before_hash": outcome.Before.Hash, "after_hash": outcome.After.Hash, "changed_files": changedFiles(outcome.Before, outcome.After), "report": outcome.Report, "evidence_file": fmt.Sprintf("round-%d-%s-implementation.json", t.Round, outcome.Domain)})
	}
	return briefs
}
