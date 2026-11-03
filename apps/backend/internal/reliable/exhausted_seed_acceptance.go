package reliable

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
)

func (e *Engine) exhaustedParentMetadata(child *Task, seed *ExhaustedSeed) (*Task, error) {
	parent, err := e.Store.Get(seed.ParentID)
	if err != nil || child.ParentTaskID != seed.ParentID || parent.SuccessorTaskID != child.ID || projectKey(parent.Project) != projectKey(child.Project) || exhaustedParentIdentity(parent) != seed.ParentIdentity {
		return nil, errors.New("batch_parent_changed: the retained batch identity changed")
	}
	if err = exhaustedRetainedParent(parent); err != nil {
		return nil, err
	}
	proofTask := *parent
	proofTask.Reason += seed.ParentReason
	if err = e.recoveryQuiescence(&proofTask); err != nil {
		return nil, err
	}
	if err = e.validateRecoveryHistory(parent); err != nil {
		return nil, fmt.Errorf("batch_seed_invalid: previous evidence changed: %w", err)
	}
	var contract Task
	if err = e.readRecoveryJSON(parent.ID, "contract.json", &contract); err != nil || !sameRecoveryContract(&contract, parent) {
		return nil, errors.New("recovery_evidence_invalid: frozen parent contract changed")
	}
	return parent, nil
}

func (e *Engine) validateSeedAcceptance(t *Task) error {
	if t.SeedPending != nil {
		return errors.New("batch_seed_invalid: retained-source validation is incomplete")
	}
	if len(t.SeedEvidence) == 0 {
		return nil
	}
	intent, err := e.publishedSeedIntent(t)
	if err != nil {
		return err
	}
	parent, err := e.exhaustedParentMetadata(t, &intent.Seed)
	if err != nil {
		return err
	}
	hash, err := e.seedAncestryDigest(parent)
	if err != nil || hash != intent.AncestryEvidenceHash || hash == "" {
		return errors.New("batch_seed_invalid: inherited provenance changed after validation")
	}
	return nil
}

func (e *Engine) seedAncestryDigest(t *Task) (string, error) {
	seen, hashes := map[string]bool{}, map[string]string{}
	for {
		if seen[t.ID] || len(seen) >= 1024 {
			return "", errors.New("batch_seed_invalid: retained-source ancestry is cyclic or exceeds its validation bound")
		}
		seen[t.ID] = true
		for _, name := range seedHistoryNames(t) {
			if !safeRelative(filepath.ToSlash(name)) {
				return "", errors.New("batch_seed_invalid: unsafe ancestor evidence reference")
			}
			raw, err := readResourceMetadata(filepath.Join(e.Root, "evidence", t.ID, name))
			if err != nil {
				return "", fmt.Errorf("batch_seed_invalid: missing ancestor evidence: %w", err)
			}
			hashes[t.ID+"/"+filepath.ToSlash(name)] = digest(raw)
		}
		if len(t.SeedEvidence) == 0 {
			break
		}
		intent, err := e.publishedSeedIntent(t)
		if err != nil {
			return "", err
		}
		t, err = e.exhaustedParentMetadata(t, &intent.Seed)
		if err != nil {
			return "", err
		}
	}
	raw, err := json.Marshal(hashes)
	return digest(raw), err
}

func seedHistoryNames(t *Task) []string {
	names := []string{"contract.json"}
	for _, list := range [][]string{t.PlanEvidence, t.DomainEvidence, t.SetupEvidence, t.RequirementEvidence, t.RecoveryEvidence, t.SeedEvidence} {
		names = append(names, list...)
	}
	if t.CheckPlanEvidence != "" {
		names = append(names, t.CheckPlanEvidence)
	}
	for i := range t.Checks {
		names = append(names, filepath.Join(fmt.Sprintf("round-%d", t.Round), fmt.Sprintf("check-%d.log", i+1)))
	}
	for _, run := range t.Runs {
		if run.Status == dispatchNotStarted {
			continue
		}
		for _, name := range []string{run.InputEvidence, run.StartEvidence, run.EndEvidence} {
			if name != "" {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}
