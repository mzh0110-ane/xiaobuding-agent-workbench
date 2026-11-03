package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const resourceRoleInstructions = `
RESOURCE INPUT CONTRACT: This candidate contains source but defers input resources, including question-bank JSON under data/artifacts. Search the line-oriented resource-paths.txt in the task EvidenceRoot for exact paths; resource-library.json holds their frozen identities. Read inventories as data, never edit them or print the entire library. Existing frozen resources are read-only; do not change/delete them, edit .git/sparse state, or access the original project's writable files. To request an input, return resource_requests with at most 64 exact relative paths, leave read_files and all other arrays empty, use empty strings (review verdict: blocked), and make no source changes in that invocation. The controller will materialize independent verified copies then start a fresh session. At most two such exchanges are allowed per role invocation. Request any inputs needed by actual integration tests before claiming completion. In a normal report return resource_requests: []. Never claim to have read an input merely because it is listed in the inventory. Missing resources and checks are unverified, not a passing result.
`

func (e *Engine) call(ctx context.Context, t *Task, role, prompt string, round int, domains ...string) (*AgentResult, error) {
	if t.Baseline == nil || t.Baseline.ResourceHash == "" {
		return e.callOnce(ctx, t, role, prompt, round, domains...)
	}
	for attempt := 0; attempt < 3; attempt++ {
		before, err := TakeSnapshot(ctx, t.Worktree)
		if err != nil {
			return nil, err
		}
		result, err := e.callOnce(ctx, t, role, prompt, round, domains...)
		if err != nil {
			return nil, err
		}
		paths, clean, err := resourceRequestOutput(result.Output)
		if err != nil {
			return nil, err
		}
		if len(paths) == 0 {
			result.Output = clean
			return result, nil
		}
		if before.ResourceHash == "" || attempt == 2 {
			return nil, errors.New("resource_request: input request budget exhausted or profile unavailable")
		}
		after, err := TakeSnapshot(ctx, t.Worktree)
		if err != nil {
			return nil, err
		}
		if before.Hash != after.Hash {
			return nil, errors.New("resource_request: source changed during a preparation-only request")
		}
		if err = materializeResources(ctx, t.Worktree, paths); err != nil {
			return nil, err
		}
		name := "resource-request-" + randomID() + ".json"
		record := map[string]any{"session": result.Session, "paths": paths, "source_hash": before.Hash, "resource_hash": before.ResourceHash, "round": round}
		if err = e.saveJSON(t.ID, name, record); err != nil {
			return nil, err
		}
		_, err = e.Store.Update(t.ID, func(saved *Task) error { saved.SetupEvidence = append(saved.SetupEvidence, name); return nil })
		if err != nil {
			return nil, err
		}
		prompt += "\nThe previous input request was materialized and verified. Reread the actual files in this fresh session, then complete your original role."
	}
	return nil, errors.New("resource_request: exhausted")
}

func resourceRequestOutput(raw []byte) ([]string, []byte, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, nil, err
	}
	field, ok := object["resource_requests"]
	if !ok {
		return nil, raw, nil
	}
	var paths []string
	if err := json.Unmarshal(field, &paths); err != nil {
		return nil, nil, err
	}
	delete(object, "resource_requests")
	if len(paths) > 64 {
		return nil, nil, errors.New("resource_request: too many requested paths")
	}
	if len(paths) > 0 {
		for key, value := range object {
			var content any
			if err := json.Unmarshal(value, &content); err != nil {
				return nil, nil, err
			}
			if key == "verdict" && content == stateBlocked {
				continue
			}
			switch item := content.(type) {
			case string:
				if item != "" {
					return nil, nil, fmt.Errorf("resource_request: request also claims completed %s", key)
				}
			case []any:
				if len(item) > 0 {
					return nil, nil, errors.New("resource_request: request also claims completed work")
				}
			default:
				return nil, nil, errors.New("resource_request: unsupported response")
			}
		}
	}
	clean, err := json.Marshal(object)
	return paths, clean, err
}
