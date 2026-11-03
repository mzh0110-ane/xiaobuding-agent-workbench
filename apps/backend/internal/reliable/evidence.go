package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func (e *Engine) acceptanceEvidence(ctx context.Context, t *Task) (string, error) {
	if t.RequirementRevision != t.AuditedRevision {
		return "", errRequirementsChanged
	}
	if t.SeedPending != nil || len(t.SeedEvidence) > 0 {
		if err := e.validateSeedAcceptance(t); err != nil {
			return "", err
		}
	}
	snap, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return "", err
	}
	if err = e.validateAutomaticCheckPlan(t); err != nil {
		return "", err
	}
	commands := verificationCommands(t)
	checksPass := len(t.Checks) == len(commands) && len(t.Checks) > 0
	for i, check := range t.Checks {
		if check.ExitCode != 0 || check.VerificationProblem != "" || i >= len(commands) || check.Command != commands[i] {
			checksPass = false
		}
	}
	decision, _ := Decide(t.Review, checksPass, snap.Hash == t.Snapshot, t.RequiredReads)
	if decision != Pass {
		return "", errors.New("candidate no longer matches its passing evidence")
	}
	evidenceHash, evidenceErr := e.evidenceDigest(t)
	if evidenceErr != nil || t.EvidenceHash == "" || evidenceHash != t.EvidenceHash {
		return "", errors.New("saved evidence no longer matches the reviewed candidate")
	}
	patch := filepath.Join(e.Root, "evidence", t.ID, fmt.Sprintf("round-%d", t.Round), "changes.patch")
	raw, err := os.ReadFile(patch)
	if err != nil {
		return "", err
	}
	if digest(raw) != t.PatchHash {
		return "", errors.New("saved patch no longer matches the reviewed candidate")
	}
	return patch, nil
}

// evidenceDigest binds the frozen contract, reports, real check logs and patch
// to the stored review decision. It detects accidental deletion or alteration;
// it is not protection against an administrator rewriting the database too.
func (e *Engine) evidenceDigest(t *Task) (string, error) {
	current, err := e.Store.Get(t.ID)
	if err != nil {
		return "", err
	}
	return e.evidenceDigestSnapshot(current)
}

// The caller may hold the store write transaction. Use only this snapshot;
// opening another store transaction here would deadlock final sealing.
func (e *Engine) evidenceDigestSnapshot(t *Task) (string, error) {
	attachments := workAttachments(t)
	if err := e.validateAttachments(attachments); err != nil {
		return "", err
	}
	round := fmt.Sprintf("round-%d", t.Round)
	names := []string{"contract.json", filepath.Join(round, "changes.patch")}
	if t.Baseline != nil && t.Baseline.ResourceHash != "" {
		names = append(names, resourceLibraryName, "resource-paths.txt")
	}
	for _, suffix := range []string{"audit", "before", "implementation", "checks", "source", "review", "decision"} {
		names = append(names, round+"-"+suffix+".json")
	}
	for i := range verificationCommands(t) {
		names = append(names, filepath.Join(round, fmt.Sprintf("check-%d.log", i+1)))
	}
	names = append(names, t.PlanEvidence...)
	names = append(names, t.DomainEvidence...)
	names = append(names, t.SetupEvidence...)
	names = append(names, t.RequirementEvidence...)
	names = append(names, t.RecoveryEvidence...)
	names = append(names, t.SeedEvidence...)
	if t.CheckPlanEvidence != "" {
		names = append(names, t.CheckPlanEvidence)
	}
	for _, d := range t.Runs {
		if d.Status != dispatchNotStarted {
			names = append(names, d.InputEvidence, d.StartEvidence)
			if d.Status != "interrupted" {
				names = append(names, d.EndEvidence)
			}
		}
	}
	hashes := make(map[string]string, len(names))
	if len(attachments) > 0 {
		raw, marshalErr := json.Marshal(attachments)
		if marshalErr != nil {
			return "", marshalErr
		}
		hashes["controller-attachment-inputs"] = digest(raw)
		for _, attachment := range attachments {
			hashes["attachment/"+attachment.ID] = attachment.SHA256
		}
	}
	if len(t.Runs) > 0 {
		raw, marshalErr := json.Marshal(t.Runs)
		if marshalErr != nil {
			return "", marshalErr
		}
		hashes["controller-dispatch-history"] = digest(raw)
	}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(e.Root, "evidence", t.ID, name))
		if err != nil {
			return "", fmt.Errorf("required evidence %s: %w", name, err)
		}
		hashes[name] = digest(raw)
	}
	if err := e.validateContextBoundEvidence(t, hashes); err != nil {
		return "", err
	}
	raw, err := json.Marshal(hashes)
	if err != nil {
		return "", err
	}
	return digest(raw), nil
}

func (e *Engine) sealReview(id string, decision Decision, reason string, revision int) error {
	_, err := e.Store.Update(id, func(saved *Task) error {
		if saved.State == stateCancelling || saved.State == stateCancelled {
			return context.Canceled
		}
		if saved.RequirementRevision != revision {
			return errRequirementsChanged
		}
		hash, hashErr := e.evidenceDigestSnapshot(saved)
		if hashErr != nil {
			return hashErr
		}
		saved.EvidenceHash = hash
		if decision == Pass {
			saved.State, saved.Reason = stateAwaitingAcceptance, reason
		}
		return nil
	})
	return err
}
