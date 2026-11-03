package reliable

import (
	"context"
	"errors"
)

func (e *Engine) validateExhaustedAncestry(ctx context.Context, t *Task) error {
	seen := map[string]bool{}
	for len(t.SeedEvidence) > 0 {
		if seen[t.ID] || len(seen) >= 1024 {
			return errors.New("batch_seed_invalid: retained-source ancestry is cyclic or exceeds its validation bound")
		}
		seen[t.ID] = true
		intent, err := e.publishedSeedIntent(t)
		if err != nil {
			return err
		}
		ancestor, err := e.exhaustedParentMetadata(t, &intent.Seed)
		if err != nil {
			return err
		}
		snap, err := e.validateRecoveryCandidate(ctx, ancestor)
		if err != nil {
			return err
		}
		if !sameSeedSource(intent.Source, snap) {
			return errors.New("batch_seed_invalid: retained-source ancestor bytes changed")
		}
		t = ancestor
	}
	return nil
}

func (e *Engine) publishedSeedIntent(t *Task) (*exhaustedSeedIntent, error) {
	seed, err := e.exhaustedSeedDescriptor(t)
	if err != nil || t.SeedPending != nil || len(t.SeedEvidence) != 2 {
		return nil, errors.New("batch_seed_invalid: incomplete ancestor inheritance evidence")
	}
	var intent exhaustedSeedIntent
	if err = e.readRecoveryJSON(t.ID, seed.Intent, &intent); err != nil || intent.Source == nil || intent.Source.Hash == "" {
		return nil, errors.New("batch_seed_invalid: ancestor retained-source intent changed")
	}
	var receipt exhaustedSeedReceipt
	if err = e.readRecoveryJSON(t.ID, t.SeedEvidence[1], &receipt); err != nil || receipt.ChildID != t.ID || receipt.ParentID != seed.ParentID || receipt.Intent != seed.Intent || receipt.SourceHash != intent.Source.Hash || receipt.Base != t.Base {
		return nil, errors.New("batch_seed_invalid: ancestor inherited-source receipt changed")
	}
	return &intent, nil
}
