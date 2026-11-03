package reliable

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type automaticCheckPlan struct {
	Commands   []string `json:"commands"`
	SourceHash string   `json:"source_hash"`
	Round      int      `json:"round"`
}

func verificationCommands(t *Task) []string {
	if len(t.Commands) > 0 {
		return t.Commands
	}
	return t.DiscoveredCommands
}

func automaticCheckInstructions(t *Task) string {
	if !t.AutoChecks {
		return ""
	}
	return "\nAUTOMATIC CHECK SETUP: No usable checks existed when this task started. Include meaningful automated assertions covering the goal, and a runnable root package.json test/build/typecheck script, go.mod tests, or tests/test*.py unittest suite. Do not use echo, exit-only or placeholder success. Shared check definitions and tests must be assigned to exactly one participating writer within the authorized scope. The controller will discover and run actual commands, then freeze the first usable check plan across rework. Preserve every existing assertion and the frozen check definitions. If the CURRENT folder has no source files, the auditor must honestly return empty read_files and required_read_files; after files exist, normal mandatory source reads apply. Independent review must inspect the generated tests and verify they cover the requested behavior."
}

func (e *Engine) ensureAutomaticChecks(t *Task, snap *Snapshot) (string, error) {
	if !t.AutoChecks {
		return "", nil
	}
	current := discoverChecks(t.Worktree)
	if len(t.DiscoveredCommands) > 0 {
		if err := e.validateAutomaticCheckPlan(t); err != nil {
			return "", err
		}
		for _, command := range t.DiscoveredCommands {
			if !slices.Contains(current, command) {
				return "automatic_checks_missing: restore the frozen check definitions and real tests", nil
			}
		}
		return "", nil
	}
	if len(current) == 0 {
		reason := "automatic_checks_missing: implement real assertions and a discoverable test, build or typecheck command before verification"
		err := e.saveJSON(t.ID, fmt.Sprintf("round-%d-check-discovery.json", t.Round), map[string]any{"reason": reason, "source_hash": snap.Hash})
		return reason, err
	}
	name := "check-plan-" + randomID() + ".json"
	plan := automaticCheckPlan{Commands: current, SourceHash: snap.Hash, Round: t.Round}
	if err := e.saveJSON(t.ID, name, plan); err != nil {
		return "", err
	}
	saved, err := e.Store.Update(t.ID, func(saved *Task) error {
		if len(saved.DiscoveredCommands) > 0 {
			return errors.New("automatic check plan was already frozen")
		}
		saved.DiscoveredCommands, saved.CheckPlanEvidence = current, name
		return nil
	})
	if err == nil {
		t.DiscoveredCommands, t.CheckPlanEvidence = saved.DiscoveredCommands, saved.CheckPlanEvidence
	}
	return "", err
}

func (e *Engine) validateAutomaticCheckPlan(t *Task) error {
	if !t.AutoChecks {
		return nil
	}
	if !safeRelative(t.CheckPlanEvidence) || strings.Contains(t.CheckPlanEvidence, "/") {
		return errors.New("automatic check plan evidence is missing")
	}
	raw, err := os.ReadFile(filepath.Join(e.Root, "evidence", t.ID, t.CheckPlanEvidence))
	if err != nil {
		return err
	}
	var plan automaticCheckPlan
	if decodeStrict(raw, &plan) != nil || len(plan.Commands) == 0 || plan.SourceHash == "" || plan.Round < 1 || !slices.Equal(plan.Commands, t.DiscoveredCommands) {
		return errors.New("automatic check plan no longer matches its frozen evidence")
	}
	return nil
}

var zeroTestsPattern = regexp.MustCompile(`(?mi)(^|\n)\s*(Ran 0 tests|# tests 0\s*$|ℹ tests 0\s*$)`)
var goTestPassPattern = regexp.MustCompile(`(?m)^ok\s+\S+`)

func automaticCheckHasNoTests(command, logPath string) bool {
	if !strings.Contains(command, "test") {
		return false
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return false
	}
	if strings.HasPrefix(command, "go test ") {
		return bytesContainNoTestPackages(raw) && !goTestPassPattern.Match(raw)
	}
	return zeroTestsPattern.Match(raw)
}

func bytesContainNoTestPackages(raw []byte) bool {
	return strings.Contains(string(raw), "[no test files]")
}
