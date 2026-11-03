package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func checkReads(read []string, snapshots ...*Snapshot) error {
	if len(read) == 0 {
		return errors.New("agent reported no source reads")
	}
	for _, path := range read {
		found := false
		for _, snap := range snapshots {
			if snap != nil && snap.AvailableResources[path] != "" {
				found = true
			}
			if snap != nil && snap.Files[path] != "" && snap.Files[path] != "deleted" {
				found = true
			}
		}
		if !safeRelative(path) || !found {
			return fmt.Errorf("read report references an unavailable source file: %s", path)
		}
	}
	return nil
}

func (e *Engine) audit(ctx context.Context, t *Task) error {
	if err := e.freezeRequirements(t); err != nil {
		return err
	}
	if err := e.setState(t.ID, stateAuditing, t.Reason); err != nil {
		return err
	}
	before, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return err
	}
	if err = e.validateAuditResumeSource(t, before); err != nil {
		return err
	}
	if before.Head != t.Base {
		return errors.New("agent changed the worktree Git history")
	}
	contextJSON, _ := json.Marshal(map[string]any{"goal": taskGoal(t), "commands": verificationCommands(t), "user_allowed_paths": t.AllowedPaths, "previous_failure": t.Reason})
	prompt := "\nROLE: READ-ONLY AUDITOR AND PLANNER. Read current code and identify the smallest sufficient implementation scope. Do not modify any file. Put only blocking missing requirements in unknowns; non-blocking uncertainty belongs in inferences. facts must cite file paths or direct evidence. required_read_files is the nonempty subset of read_files containing essential current code, tests or contracts that the implementer MUST freshly read. Auxiliary files you merely consulted are not automatically mandatory. allowed_paths must be concrete relative files/directories, or '.' only when genuinely necessary. Plan a minimal implementation and a lightweight documentation impact check. The controller has frozen these inputs:\n" + string(contextJSON)
	prompt += domainAuditInstructions(t.ModelPlan)
	prompt += automaticCheckInstructions(t)
	result, err := e.callAuditWithRetry(ctx, t, prompt, before)
	if err != nil {
		return err
	}
	after, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return err
	}
	if before.Hash != after.Hash {
		return errors.New("read-only audit changed the source")
	}
	var audit Audit
	if err = decodeStrict(result.Output, &audit); err != nil {
		return err
	}
	if err = validateTaskAudit(t, &audit, after); err != nil {
		return err
	}
	if err = validateDomainAudit(t.ModelPlan, &audit); err != nil {
		return err
	}
	return e.commitAudit(ctx, t, &audit)
}

func validateAudit(audit *Audit, after *Snapshot, userScope []string) error {
	if err := checkReads(audit.ReadFiles, after); err != nil {
		return err
	}
	if err := checkReads(audit.RequiredReadFiles, after); err != nil {
		return err
	}
	for _, path := range audit.RequiredReadFiles {
		if !containsPath(audit.ReadFiles, path) {
			return errors.New("mandatory read was not established by the auditor")
		}
	}
	return validateAuditPlan(audit, userScope)
}

func validateTaskAudit(t *Task, audit *Audit, snap *Snapshot) error {
	if t.AutoChecks && len(snap.Files) == 0 {
		if len(audit.ReadFiles) > 0 || len(audit.RequiredReadFiles) > 0 {
			return errors.New("empty source audit must not invent file reads")
		}
		return validateAuditPlan(audit, t.AllowedPaths)
	}
	return validateAudit(audit, snap, t.AllowedPaths)
}

func validateAuditPlan(audit *Audit, userScope []string) error {
	if err := normalizeAuditScopes(audit); err != nil {
		return err
	}
	if audit.Summary == "" || len(audit.AllowedPaths) == 0 || len(audit.Plan) == 0 {
		return errors.New("audit plan is incomplete")
	}
	if len(audit.Unknowns) > 0 {
		return fmt.Errorf("clarification required: %s", strings.Join(audit.Unknowns, "; "))
	}
	for _, path := range audit.AllowedPaths {
		if path != "." && !safeRelative(path) {
			return errors.New("audit proposed an unsafe scope")
		}
		if len(userScope) > 0 && !withinScope(path, userScope) {
			return errors.New("audit exceeded the user-authorized scope")
		}
	}
	return nil
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}
func unionPaths(a, b []string) []string {
	set := make(map[string]bool)
	for _, p := range a {
		set[p] = true
	}
	for _, p := range b {
		set[p] = true
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func withinScope(path string, allowed []string) bool {
	for _, prefix := range allowed {
		if prefix == "." || path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

type implementation struct {
	Summary       string   `json:"summary"`
	ReadFiles     []string `json:"read_files"`
	DocsImpact    string   `json:"docs_impact"`
	ResidualRisks []string `json:"residual_risks"`
}

func (e *Engine) implement(ctx context.Context, t *Task) error {
	if t.ModelPlan.Mode != "" {
		return e.implementDomains(ctx, t)
	}
	if err := e.saveJSON(t.ID, fmt.Sprintf("round-%d-audit.json", t.Round), t.Audit); err != nil {
		return err
	}
	before, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return err
	}
	if err = e.saveJSON(t.ID, fmt.Sprintf("round-%d-before.json", t.Round), before); err != nil {
		return err
	}
	data, _ := json.Marshal(map[string]any{"goal": taskGoal(t), "audit": t.Audit, "commands": verificationCommands(t), "previous_failure": t.Reason, "round": t.Round, "max_rounds": MaxRounds, "previous_check_directory": filepath.Join(e.Root, "evidence", t.ID, fmt.Sprintf("round-%d", t.Round-1)), "previous_review_file": filepath.Join(e.Root, "evidence", t.ID, fmt.Sprintf("round-%d-review.json", t.Round-1))})
	prompt := "\nROLE: IMPLEMENTER. You are the only source writer. You MUST freshly read every path in audit.required_read_files before implementing and include each in your final read_files acknowledgement. The other audit.read_files entries are supporting evidence, not mandatory rereads. Implement the goal only within the current audit.allowed_paths. Preserve existing tests and their intent; do not weaken checks to make them pass. Address the previous failure, inspecting previous logs if needed. Perform a lightweight documentation impact check. Do not modify source outside this worktree. Inputs:\n" + string(data)
	prompt += automaticCheckInstructions(t)
	result, err := e.call(ctx, t, roleImplementer, prompt, t.Round)
	if err != nil {
		return err
	}
	after, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return err
	}
	if after.Head != t.Base {
		return errors.New("implementation changed Git history")
	}
	if t.Baseline == nil {
		return errors.New("original source evidence is missing")
	}
	if err = checkScope(changedFiles(t.Baseline, after), t.ApprovedPaths); err != nil {
		return err
	}
	if err = checkScope(changedFiles(before, after), t.Audit.AllowedPaths); err != nil {
		return err
	}
	var report implementation
	if err = decodeStrict(result.Output, &report); err != nil {
		return err
	}
	if err = checkReads(report.ReadFiles, before, after); err != nil {
		return err
	}
	if err = validateImplementation(report, t.Audit.RequiredReadFiles); err != nil {
		return err
	}
	return e.saveJSON(t.ID, fmt.Sprintf("round-%d-implementation.json", t.Round), report)
}

func validateImplementation(report implementation, requiredReads []string) error {
	if report.Summary == "" || report.DocsImpact == "" {
		return errors.New("implementation report is incomplete")
	}
	for _, required := range requiredReads {
		found := false
		for _, path := range report.ReadFiles {
			if path == required {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("implementation omitted required read: %s", required)
		}
	}
	return nil
}

func (e *Engine) verifyAndReview(ctx context.Context, t *Task) (Decision, string, error) {
	if err := e.setState(t.ID, stateChecking, ""); err != nil {
		return Block, "", err
	}
	before, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return Block, "", err
	}
	if reason, setupErr := e.ensureAutomaticChecks(t, before); setupErr != nil {
		return Block, "", setupErr
	} else if reason != "" {
		return Rework, reason, nil
	}
	roundDir := filepath.Join(e.Root, "evidence", t.ID, fmt.Sprintf("round-%d", t.Round))
	if err = os.MkdirAll(roundDir, 0700); err != nil {
		return Block, "", err
	}
	checks, passed, err := e.executeVerification(ctx, t, roundDir)
	if err != nil {
		return Block, "", err
	}
	after, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return Block, "", err
	}
	if before.Hash != after.Hash {
		return Block, "Verification modified source or generated unignored files", nil
	}
	if _, err = e.Store.Update(t.ID, func(saved *Task) error { saved.Checks = checks; return nil }); err != nil {
		return Block, "", err
	}
	if err = e.saveJSON(t.ID, fmt.Sprintf("round-%d-checks.json", t.Round), checks); err != nil {
		return Block, "", err
	}
	if !passed {
		return Rework, "A required verification command failed; inspect this round's check logs", nil
	}
	return e.review(ctx, t, after, checks, roundDir)
}

func (e *Engine) executeVerification(ctx context.Context, t *Task, roundDir string) ([]Check, bool, error) {
	commands := verificationCommands(t)
	checks := make([]Check, 0, len(commands))
	passed := len(commands) > 0
	for i, command := range commands {
		logPath := filepath.Join(roundDir, fmt.Sprintf("check-%d.log", i+1))
		check, err := runCheck(observeCheckProcess(ctx, command), t.Worktree, command, logPath, e.Redact)
		if err != nil {
			return nil, false, err
		}
		if t.AutoChecks && automaticCheckHasNoTests(command, logPath) {
			check.VerificationProblem = "The runner executed zero tests; no behavior was verified"
		}
		checks = append(checks, check)
		if check.ExitCode != 0 || check.VerificationProblem != "" {
			passed = false
		}
		if err = ctx.Err(); err != nil {
			return nil, false, err
		}
	}
	return checks, passed, nil
}

func requiredReviewFiles(t *Task, snap *Snapshot) []string {
	set := make(map[string]bool)
	for _, p := range changedFiles(t.Baseline, snap) {
		set[p] = true
	}
	for _, p := range t.Audit.RequiredReadFiles {
		set[p] = true
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func (e *Engine) review(ctx context.Context, t *Task, snap *Snapshot, checks []Check, roundDir string) (Decision, string, error) {
	patch := filepath.Join(roundDir, "changes.patch")
	if err := ExportPatch(ctx, t.Worktree, t.Base, patch); err != nil {
		return Block, "", err
	}
	patchBytes, err := os.ReadFile(patch)
	if err != nil {
		return Block, "", err
	}
	if err = e.setState(t.ID, stateReviewing, ""); err != nil {
		return Block, "", err
	}
	required := requiredReviewFiles(t, snap)
	data, _ := json.Marshal(map[string]any{"goal": taskGoal(t), "audit": t.Audit, "required_read_files": required, "patch_file": patch, "source_hash": snap.Hash, "checks": checks, "check_log_directory": roundDir, "round": t.Round})
	prompt := "\nROLE: INDEPENDENT READ-ONLY REVIEWER. You are a fresh session, independent of implementation. Do not modify or fix any file. Read the actual diff, required source files and the raw verification logs. Judge requirement coverage, behavior, regressions, scope and documentation impact. Treat implementation claims as unverified. Passing command exit codes are not proof that tests are meaningful: inspect changed tests and check definitions for weakened assertions or placeholder commands. Return 'blocked' if necessary evidence is missing, 'rework' for P0/P1 or unmet requirements, 'pass' only when the candidate meets the goal. P2/P3 suggestions alone do not require rework. Inputs:\n" + string(data)
	result, err := e.call(ctx, t, roleReviewer, prompt, t.Round)
	if err != nil {
		return Block, "", err
	}
	after, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return Block, "", err
	}
	decision, reason := Decide(result.Output, true, snap.Hash == after.Hash, required)
	if decision != Block {
		var report Review
		if err = decodeStrict(result.Output, &report); err != nil {
			return Block, "", err
		}
		if err = checkReads(report.ReadFiles, t.Baseline, snap); err != nil {
			decision = Block
			reason = err.Error()
		}
	}
	if _, err = e.Store.Update(t.ID, func(saved *Task) error {
		saved.Review = json.RawMessage(result.Output)
		saved.Snapshot = snap.Hash
		saved.RequiredReads = required
		saved.PatchHash = digest(patchBytes)
		return nil
	}); err != nil {
		return Block, "", err
	}
	if err = e.saveJSON(t.ID, fmt.Sprintf("round-%d-source.json", t.Round), snap); err != nil {
		return Block, "", err
	}
	if err = e.saveJSON(t.ID, fmt.Sprintf("round-%d-review.json", t.Round), json.RawMessage(result.Output)); err != nil {
		return Block, "", err
	}
	if err = e.saveJSON(t.ID, fmt.Sprintf("round-%d-decision.json", t.Round), map[string]any{"decision": decision, "reason": reason, "snapshot": snap.Hash, "review_session": result.Session}); err != nil {
		return Block, "", err
	}
	if err = e.sealReview(t.ID, decision, reason, t.AuditedRevision); err != nil {
		return Block, "", err
	}
	return decision, reason, nil
}
