package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type handoffInput struct {
	SharedContext       *SharedContextReference `json:"shared_context,omitempty"`
	SharedContextPrompt string                  `json:"shared_context_prompt,omitempty"`
	ProjectRecord       *ProjectRecord          `json:"project_record,omitempty"`
	ProjectContext      string                  `json:"project_context,omitempty"`
	ProjectEvidence     string                  `json:"project_evidence,omitempty"`
	Attachments         []Attachment            `json:"attachments,omitempty"`
	RequirementRevision int                     `json:"requirement_revision,omitempty"`
	TaskID              string                  `json:"task_id"`
	Goal                string                  `json:"goal"`
	Commands            []string                `json:"commands"`
	SourceHash          string                  `json:"source_hash"`
	Base                string                  `json:"base"`
	RequiredReads       []string                `json:"required_reads"`
	Audit               *Audit                  `json:"audit,omitempty"`
	Checks              []Check                 `json:"checks"`
	PreviousFailure     string                  `json:"previous_failure"`
	EvidenceRoot        string                  `json:"evidence_root"`
	Dispatch            Dispatch                `json:"dispatch"`
}

func (e *Engine) beginDispatch(ctx context.Context, t *Task, role string, round int, domains ...string) (*Dispatch, []byte, error) {
	projectRecord, err := e.Project(t.ID)
	if err != nil {
		return nil, nil, err
	}
	snap, err := TakeSnapshot(ctx, t.Worktree)
	if err != nil {
		return nil, nil, err
	}
	shared, err := e.prepareDispatchContext(ctx, t, snap)
	if err != nil {
		return nil, nil, err
	}
	d := Dispatch{ContextHash: shared.Reference.IndexSHA256, ID: randomID(), Role: role, Round: round, SourceHash: snap.Hash, Status: "preparing", Started: time.Now().UTC().Format(time.RFC3339Nano)}
	if len(domains) > 0 {
		d.Domain = domains[0]
	}
	d.InputEvidence = "dispatch-" + d.ID + "-input.json"
	d.StartEvidence = "dispatch-" + d.ID + "-start.json"
	d.EndEvidence = "dispatch-" + d.ID + "-result.json"
	current, err := e.Store.Update(t.ID, func(saved *Task) error {
		if saved.State == stateCancelled || saved.State == stateCancelling {
			return context.Canceled
		}
		plan := saved.ModelPlan
		if plan == (ModelPlan{}) {
			plan = e.defaultPlan()
		}
		id, model, effort := plan.selection(role, d.Domain)
		provider, selectedModel, selectedEffort, lookupErr := e.resolveSelection(id, model, effort)
		if lookupErr != nil {
			return lookupErr
		}
		d.ProviderID, d.ProviderName, d.Model, d.Effort = provider.ID, provider.Name, selectedModel, selectedEffort
		d.PlanRevision = saved.PlanRevision
		saved.Runs = append(saved.Runs, d)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	input := handoffInput{RequirementRevision: t.AuditedRevision, TaskID: t.ID, Goal: taskGoal(t), Commands: verificationCommands(current), Base: current.Base, SourceHash: snap.Hash, RequiredReads: current.RequiredReads, Audit: current.Audit, Checks: current.Checks, PreviousFailure: current.Reason, EvidenceRoot: filepath.Join(e.Root, "evidence", t.ID), Dispatch: d}
	input.Attachments = workAttachments(t)
	input.SharedContext, input.SharedContextPrompt = &shared.Reference, shared.Prompt
	input.ProjectRecord = projectRecord
	input.ProjectContext = compactProjectContext(projectRecord)
	input.ProjectEvidence = filepath.Join(e.Root, "evidence", t.ID, d.InputEvidence)
	if current.Audit != nil {
		input.RequiredReads = unionPaths(input.RequiredReads, current.Audit.RequiredReadFiles)
	}
	if err = e.saveJSON(t.ID, d.StartEvidence, d); err != nil {
		_ = e.markDispatch(t.ID, &d, dispatchNotStarted)
		return nil, nil, err
	}
	if err = e.saveJSON(t.ID, d.InputEvidence, input); err != nil {
		_ = e.markDispatch(t.ID, &d, dispatchNotStarted)
		return nil, nil, err
	}
	if err = e.markDispatch(t.ID, &d, "running"); err != nil {
		return nil, nil, err
	}
	input.ProjectRecord = nil // Full history remains in the immutable input evidence.
	raw, err := json.Marshal(input)
	return &d, raw, err
}

func (e *Engine) markDispatch(id string, d *Dispatch, status string) error {
	_, err := e.Store.Update(id, func(saved *Task) error {
		d.Status = status
		if status == dispatchNotStarted {
			d.EndEvidence = ""
			d.Completed = time.Now().UTC().Format(time.RFC3339Nano)
		}
		for i := range saved.Runs {
			if saved.Runs[i].ID == d.ID {
				saved.Runs[i] = *d
				return nil
			}
		}
		return errors.New("dispatch reservation is missing")
	})
	return err
}

func (e *Engine) callOnce(ctx context.Context, t *Task, role, prompt string, round int, domains ...string) (*AgentResult, error) {
	d, input, err := e.beginDispatch(ctx, t, role, round, domains...)
	if err != nil {
		return nil, err
	}
	if err = e.readSharedContextInput(t.ID, d.InputEvidence, d.ContextHash, nil); err != nil {
		e.failDispatch(t.ID, d, err)
		return nil, err
	}
	outDir := filepath.Join(e.Root, "evidence", t.ID, fmt.Sprintf("round-%d", round), role+"-"+d.ID)
	request := RoleRequest{Role: role, Directory: t.Worktree, ProviderID: d.ProviderID, Model: d.Model, Effort: d.Effort, Domain: d.Domain, WorkMode: t.ModelPlan.Mode, OutDir: outDir, Prompt: roleRules + "\nCONTROLLER HANDOFF (shared_context_prompt identifies the complete retained history): " + string(input) + prompt}
	if err = e.attachRequest(&request, workAttachments(t)); err != nil {
		e.failDispatch(t.ID, d, err)
		return nil, err
	}
	if t.Baseline != nil && t.Baseline.ResourceHash != "" {
		request.ResourceMode = true
		request.Prompt += resourceRoleInstructions
	}
	ctx = scopeActivity(ctx, d)
	emitActivity(ctx, "agent", "Agent 开始执行", d.ProviderName+" · "+d.Model)
	result, err := e.Runner.Run(ctx, request)
	err = errors.Join(err, e.readSharedContextInput(t.ID, d.InputEvidence, d.ContextHash, nil))
	if err == nil && (result == nil || strings.TrimSpace(result.Session) == "" || len(result.Output) == 0) {
		err = errors.New("agent returned no identified report")
	}
	if err != nil {
		e.failDispatch(t.ID, d, err)
		return nil, err
	}
	if err = e.completeDispatch(t.ID, d, result); err != nil {
		e.failDispatch(t.ID, d, err)
		return nil, err
	}
	return result, nil
}

func (e *Engine) completeDispatch(id string, d *Dispatch, result *AgentResult) error {
	_, err := e.Store.Update(id, func(saved *Task) error {
		for _, session := range saved.Sessions {
			if session == result.Session {
				return errors.New("agent reused an existing session; independent review is not established")
			}
		}
		d.Session, d.Status, d.Completed = result.Session, "completed", time.Now().UTC().Format(time.RFC3339Nano)
		if err := e.saveJSON(id, d.EndEvidence, map[string]any{"dispatch": d, "report": json.RawMessage(result.Output)}); err != nil {
			return err
		}
		for i := range saved.Runs {
			if saved.Runs[i].ID == d.ID {
				saved.Runs[i] = *d
			}
		}
		saved.Sessions = append(saved.Sessions, result.Session)
		return nil
	})
	return err
}

func (e *Engine) failDispatch(id string, d *Dispatch, runErr error) {
	_, _ = e.Store.Update(id, func(saved *Task) error {
		d.Status, d.Completed = "failed", time.Now().UTC().Format(time.RFC3339Nano)
		_ = e.saveJSON(id, d.EndEvidence, map[string]any{"dispatch": d, "error": runErr.Error()})
		for i := range saved.Runs {
			if saved.Runs[i].ID == d.ID {
				saved.Runs[i] = *d
			}
		}
		return nil
	})
}
