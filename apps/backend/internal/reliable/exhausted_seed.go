package reliable

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

type exhaustedSeedIntent struct {
	ChildID              string        `json:"child_id"`
	Seed                 ExhaustedSeed `json:"seed"`
	Source               *Snapshot     `json:"source"`
	AncestryEvidenceHash string        `json:"ancestry_evidence_hash"`
}

type exhaustedSeedReceipt struct {
	ChildID    string `json:"child_id"`
	ParentID   string `json:"parent_id"`
	Intent     string `json:"intent"`
	SourceHash string `json:"source_hash"`
	Base       string `json:"base"`
	Time       string `json:"time"`
}

func (e *Engine) prepareExhaustedSeed(ctx context.Context, t *Task) error {
	if t.SeedPending != nil {
		if err := e.recoveryQuiescence(t); err != nil {
			return err
		}
	}
	seed, err := e.exhaustedSeedDescriptor(t)
	if err != nil {
		return err
	}
	parentCtx, release, err := e.reserve(ctx, seed.ParentID)
	if err != nil {
		return fmt.Errorf("batch_parent_active: wait for the previous batch to become idle: %w", err)
	}
	defer release()
	parent, snap, err := e.validateExhaustedParent(parentCtx, t, seed)
	if err != nil {
		return err
	}
	intent, err := e.ensureExhaustedIntent(t, seed, snap)
	if err != nil {
		return err
	}
	if t.SeedPending == nil {
		return e.validatePublishedSeed(parentCtx, t, intent)
	}
	emitActivity(ctx, "phase", "正在承接上一批的修改", "保留原副本和审查记录；新一批从零轮开始，仍需实际检查和独立审查。")
	if err = e.ensureSeedModelPlan(t, seed); err != nil {
		return err
	}
	if err = e.validateSeedDestination(parentCtx, parent, t); err != nil {
		return err
	}
	if err = e.prepare(parentCtx, t); err != nil {
		return err
	}
	if err = e.recoveryOwnership(parentCtx, t); err != nil {
		return fmt.Errorf("recovery_candidate_invalid: %w", err)
	}
	if err = validateSeedBaseline(parent, t); err != nil {
		return err
	}
	if err = e.copyExhaustedSource(parentCtx, parent, t, intent.Source); err != nil {
		return err
	}
	return e.publishExhaustedSeed(parentCtx, t, intent)
}

func (e *Engine) validateSeedDestination(ctx context.Context, parent, child *Task) error {
	candidate := *child
	if candidate.Worktree == "" {
		candidate.Worktree, candidate.Base = filepath.Join(e.Root, "worktrees", child.ID), parent.Base
		if _, err := os.Lstat(candidate.Worktree); os.IsNotExist(err) {
			return nil
		}
	}
	if err := e.recoveryOwnership(ctx, &candidate); err != nil {
		return fmt.Errorf("recovery_candidate_invalid: %w", err)
	}
	return nil
}

func (e *Engine) exhaustedSeedDescriptor(t *Task) (*ExhaustedSeed, error) {
	if t.SeedPending != nil {
		copy := *t.SeedPending
		return &copy, nil
	}
	if len(t.SeedEvidence) != 2 {
		return nil, errors.New("batch_seed_invalid: incomplete retained-source evidence")
	}
	var intent exhaustedSeedIntent
	if err := e.readRecoveryJSON(t.ID, t.SeedEvidence[0], &intent); err != nil || intent.ChildID != t.ID || intent.Seed.Intent != t.SeedEvidence[0] {
		return nil, errors.New("batch_seed_invalid: retained-source intent is missing or changed")
	}
	return &intent.Seed, nil
}

func (e *Engine) validateExhaustedParent(ctx context.Context, child *Task, seed *ExhaustedSeed) (*Task, *Snapshot, error) {
	parent, err := e.exhaustedParentMetadata(child, seed)
	if err != nil {
		return nil, nil, err
	}
	snap, err := e.validateRecoveryCandidate(ctx, parent)
	if err != nil {
		return nil, nil, err
	}
	if seed.ExpectedSourceHash != "" && seed.ExpectedSourceHash != snap.Hash {
		return nil, nil, errors.New("batch_parent_changed: previous reviewed source changed before continuation")
	}
	if err = e.validateExhaustedAncestry(ctx, parent); err != nil {
		return nil, nil, err
	}
	return parent, snap, nil
}

func (e *Engine) ensureExhaustedIntent(t *Task, seed *ExhaustedSeed, snap *Snapshot) (*exhaustedSeedIntent, error) {
	parent, err := e.exhaustedParentMetadata(t, seed)
	if err != nil {
		return nil, err
	}
	ancestry, err := e.seedAncestryDigest(parent)
	if err != nil {
		return nil, err
	}
	want := &exhaustedSeedIntent{ChildID: t.ID, Seed: *seed, Source: snap, AncestryEvidenceHash: ancestry}
	var saved exhaustedSeedIntent
	err = e.readRecoveryJSON(t.ID, seed.Intent, &saved)
	if os.IsNotExist(err) && t.SeedPending != nil && t.Worktree == "" {
		if err = e.saveJSON(t.ID, seed.Intent, want); err == nil {
			return want, nil
		}
	}
	if err != nil || saved.ChildID != want.ChildID || saved.Seed != want.Seed || saved.AncestryEvidenceHash != ancestry || !sameSeedSource(saved.Source, want.Source) {
		return nil, errors.New("batch_seed_invalid: retained-source intent changed or disappeared")
	}
	return &saved, nil
}

func sameSeedSource(a, b *Snapshot) bool {
	return a != nil && b != nil && a.Hash == b.Hash && a.Head == b.Head && a.ResourceHash == b.ResourceHash && reflect.DeepEqual(a.Files, b.Files)
}

func (e *Engine) ensureSeedModelPlan(t *Task, seed *ExhaustedSeed) error {
	if len(t.PlanEvidence) == 0 {
		return errors.New("batch_seed_invalid: initial assignment receipt is missing")
	}
	var saved ModelPlan
	err := e.readRecoveryJSON(t.ID, t.PlanEvidence[0], &saved)
	if os.IsNotExist(err) {
		return e.saveJSON(t.ID, t.PlanEvidence[0], seed.InitialPlan)
	}
	if err != nil || saved != seed.InitialPlan {
		return errors.New("batch_seed_invalid: initial assignment receipt changed")
	}
	return nil
}

func validateSeedBaseline(parent, child *Task) error {
	if child.Base != parent.Base || child.Baseline == nil || child.Baseline.Hash != parent.Baseline.Hash ||
		child.Baseline.ResourceHash != parent.Baseline.ResourceHash || !reflect.DeepEqual(child.Baseline.Files, parent.Baseline.Files) {
		return errors.New("batch_original_changed: new worktree differs from the original frozen baseline")
	}
	return nil
}

func (e *Engine) publishExhaustedSeed(ctx context.Context, t *Task, intent *exhaustedSeedIntent) error {
	snap, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil || snap.Hash != intent.Source.Hash {
		return errors.New("batch_seed_invalid: inherited candidate does not match the retained source")
	}
	name := "seed-" + randomID() + "-validated.json"
	receipt := exhaustedSeedReceipt{ChildID: t.ID, ParentID: intent.Seed.ParentID, Intent: intent.Seed.Intent, SourceHash: snap.Hash, Base: t.Base, Time: time.Now().UTC().Format(time.RFC3339Nano)}
	if err = e.saveJSON(t.ID, name, receipt); err != nil {
		return err
	}
	updated, err := e.Store.Update(t.ID, func(saved *Task) error {
		if ctx.Err() != nil || saved.State == stateCancelled || saved.State == stateCancelling {
			return context.Canceled
		}
		if saved.SeedPending == nil || *saved.SeedPending != intent.Seed || saved.Round != 0 {
			return errors.New("batch_seed_invalid: batch changed before inherited source publication")
		}
		saved.SeedEvidence = []string{intent.Seed.Intent, name}
		saved.SeedPending = nil
		saved.State, saved.Reason = stateAuditing, "Retained changes inherited; fresh audit required for this batch"
		return nil
	})
	if err == nil {
		*t = *updated
	}
	return err
}

func (e *Engine) validatePublishedSeed(ctx context.Context, t *Task, intent *exhaustedSeedIntent) error {
	var receipt exhaustedSeedReceipt
	if err := e.readRecoveryJSON(t.ID, t.SeedEvidence[1], &receipt); err != nil || receipt.ChildID != t.ID || receipt.ParentID != intent.Seed.ParentID || receipt.Intent != intent.Seed.Intent || receipt.SourceHash != intent.Source.Hash || receipt.Base != t.Base {
		return errors.New("batch_seed_invalid: inherited-source receipt is missing or changed")
	}
	if err := e.recoveryQuiescence(t); err != nil {
		return err
	}
	if err := e.validateRecoveryHistory(t); err != nil {
		return fmt.Errorf("batch_seed_invalid: %w", err)
	}
	_, err := e.validateRecoveryCandidate(ctx, t)
	return err
}
