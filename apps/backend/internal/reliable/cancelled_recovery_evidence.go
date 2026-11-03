package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
)

func (e *Engine) readRecoveryJSON(id, name string, value any) error {
	if !safeRelative(name) || strings.Contains(name, "/") {
		return errors.New("invalid recovery evidence reference")
	}
	raw, err := readResourceMetadata(filepath.Join(e.Root, "evidence", id, name))
	if err != nil {
		return err
	}
	return decodeStrict(raw, value)
}

func (e *Engine) validateRecoveryHistory(t *Task) error {
	if err := e.validateRecoveryInputHistory(t); err != nil {
		return err
	}
	for _, name := range append(slices.Clone(t.SetupEvidence), t.DomainEvidence...) {
		var value json.RawMessage
		if err := e.readRecoveryJSON(t.ID, name, &value); err != nil {
			return err
		}
	}
	for _, name := range t.RecoveryEvidence {
		var receipt recoveryReceipt
		if err := e.readRecoveryJSON(t.ID, name, &receipt); err != nil || receipt.TaskID != t.ID || projectKey(receipt.Project) != projectKey(t.Project) || receipt.Round > t.Round {
			return errors.New("recovery receipt is missing or inconsistent")
		}
	}
	if t.RecoveryPending != nil {
		var intent recoveryReceipt
		if err := e.readRecoveryJSON(t.ID, t.RecoveryPending.Intent, &intent); err != nil || intent.TaskID != t.ID || intent.Attempt != t.RecoveryPending.ID || intent.Round != t.RecoveryPending.Round {
			return errors.New("recovery intent is missing or inconsistent")
		}
	}
	return e.recoveryDispatchHistory(t)
}

func (e *Engine) validateRecoveryInputHistory(t *Task) error {
	if err := e.validateAttachments(workAttachments(t)); err != nil {
		return err
	}
	if err := e.recoveryModelHistory(t); err != nil {
		return err
	}
	return e.recoveryRequirements(t)
}

func (e *Engine) recoveryModelHistory(t *Task) error {
	if len(t.PlanEvidence) != t.PlanRevision {
		return errors.New("model-plan evidence revision count changed")
	}
	var last ModelPlan
	for i, name := range t.PlanEvidence {
		if !strings.HasPrefix(name, fmt.Sprintf("model-plan-%d-", i+1)) {
			return errors.New("model-plan evidence order changed")
		}
		if err := e.readRecoveryJSON(t.ID, name, &last); err != nil {
			return err
		}
		if last.Mode != t.ModelPlan.Mode || last.FrontendPaths != t.ModelPlan.FrontendPaths || last.BackendPaths != t.ModelPlan.BackendPaths {
			return errors.New("model-plan domains or scope changed")
		}
	}
	if last != t.ModelPlan {
		return errors.New("current assignment differs from its saved evidence")
	}
	if t.FuturePlan != nil {
		root := filepath.Join(e.Root, "future-models", t.ID)
		resolved, err := filepath.EvalSymlinks(t.FuturePlan.Evidence)
		if err != nil || !pathInside(root, resolved) {
			return errors.New("future-model evidence path changed")
		}
		raw, err := readResourceMetadata(resolved)
		var future FutureModelPlan
		if err != nil || decodeStrict(raw, &future) != nil || future != *t.FuturePlan {
			return errors.New("future-model evidence changed")
		}
	}
	return nil
}

func (e *Engine) recoveryRequirements(t *Task) error {
	if err := e.validateAuditReplanEvidence(t, nil); err != nil {
		return err
	}
	seen, revision := make(map[string]bool), 0
	for _, name := range t.RequirementEvidence {
		if seen[name] {
			return errors.New("requirement evidence reference is duplicated")
		}
		seen[name] = true
		if strings.HasPrefix(name, "audit-replan-") {
			continue
		}
		revision++
		var receipt TaskMessage
		if err := e.readRecoveryJSON(t.ID, name, &receipt); err != nil || receipt.Kind != messageRequirement || receipt.Revision != revision {
			return errors.New("requirement evidence is missing or out of order")
		}
		found := false
		for _, message := range t.Messages {
			if sameRequirementMessage(&message, &receipt) {
				found = true
			}
		}
		if !found {
			return errors.New("requirement message differs from its saved receipt")
		}
	}
	if revision != t.RequirementRevision {
		return errors.New("requirement evidence revision count changed")
	}
	return nil
}

func sameRequirementMessage(message, receipt *TaskMessage) bool {
	return message.ID == receipt.ID && message.Kind == receipt.Kind && message.Content == receipt.Content && message.Revision == receipt.Revision && message.Created == receipt.Created && reflect.DeepEqual(message.Attachments, receipt.Attachments) && reflect.DeepEqual(message.ContextAttachments, receipt.ContextAttachments)
}

func (e *Engine) recoveryDispatchHistory(t *Task) error {
	for _, run := range t.Runs {
		if err := e.recoveryDispatch(t.ID, run); err != nil {
			return err
		}
	}
	if t.EvidenceHash != "" {
		hash, err := e.evidenceDigestSnapshot(t)
		if err != nil || hash != t.EvidenceHash {
			return errors.New("previous sealed evidence changed")
		}
	}
	return nil
}

func (e *Engine) recoveryDispatch(id string, run Dispatch) error {
	if run.Status == dispatchNotStarted {
		return nil
	}
	var start Dispatch
	if err := e.readRecoveryJSON(id, run.StartEvidence, &start); err != nil || start.ID != run.ID || start.SourceHash != run.SourceHash || start.Role != run.Role || start.Round != run.Round || start.Model != run.Model || start.ProviderID != run.ProviderID {
		return errors.New("dispatch identity evidence changed")
	}
	for _, name := range []string{run.InputEvidence, run.EndEvidence} {
		if name == "" && (run.Status == "running" || run.Status == "interrupted" || run.Status == "preparing") {
			continue
		}
		var value json.RawMessage
		if err := e.readRecoveryJSON(id, name, &value); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) recoveryResources(ctx context.Context, t *Task) error {
	manifest, marker, err := loadResourceProfile(ctx, t.Worktree)
	if err != nil {
		return err
	}
	if t.Baseline.ResourceHash == "" {
		if marker != nil {
			return errors.New("candidate acquired an unexpected resource profile")
		}
		return nil
	}
	if marker == nil || manifest == nil || marker.Digest != t.Baseline.ResourceHash || marker.Base != t.Base || projectKey(marker.Manifest) != projectKey(filepath.Join(e.Root, "evidence", t.ID, resourceLibraryName)) {
		return errors.New("frozen resource identity changed")
	}
	var intent resourceMarker
	if err = e.readRecoveryJSON(t.ID, "resource-preparation.json", &intent); err != nil || !reflect.DeepEqual(&intent, marker) {
		return errors.New("resource preparation evidence changed")
	}
	path, err := profileMarkerPath(ctx, t.Worktree)
	if err != nil {
		return err
	}
	raw, err := readResourceMetadata(path + ".ready")
	var ready struct {
		Ready bool `json:"ready"`
	}
	if err != nil || decodeStrict(raw, &ready) != nil || !ready.Ready {
		return errors.New("resource checkout initialization is incomplete")
	}
	return e.recoveryResourceIndex(t, manifest)
}

func (e *Engine) recoveryResourceIndex(t *Task, manifest *resourceManifest) error {
	paths := make([]string, 0, len(manifest.Resources))
	for name := range manifest.Resources {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	raw, err := readResourceMetadata(filepath.Join(e.Root, "evidence", t.ID, "resource-paths.txt"))
	if err != nil || string(raw) != strings.Join(paths, "\n")+"\n" {
		return errors.New("resource path index is missing or changed")
	}
	return nil
}
