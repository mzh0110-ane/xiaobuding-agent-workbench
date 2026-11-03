package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

func (e *Engine) inspectProject(ctx context.Context, t *Task) (runErr error) {
	if err := readOnlyProjectBoundary(t.Project); err != nil {
		return err
	}
	if err := e.setState(t.ID, stateAuditing, ""); err != nil {
		return err
	}
	emitActivity(ctx, "phase", "只读了解项目", "直接读取所选文件夹；不创建 Git、不复制工作副本、不执行修改和验收应用。")
	d, err := e.beginInspection(t)
	if err != nil {
		return err
	}
	ctx = scopeActivity(ctx, d)
	defer func() {
		if runErr != nil {
			e.failDispatch(t.ID, d, runErr)
		}
	}()
	emitActivity(ctx, "agent", "开始只读调查", d.ProviderName+" · "+d.Model)
	out := filepath.Join(e.Root, "evidence", t.ID, "inspection-"+d.ID)
	prompt := roleRules + "\nREAD-ONLY PROJECT INSPECTION. Explain what this project does and its current progress in Simplified Chinese. Read source and documentation directly in the chosen directory. Do not edit, commit, install, run tests, start servers, or create another worktree. Separate facts, inferences and unknowns. Cite actual relative source paths in facts and read_files. Put suggested next steps in plan, but execute none.\nUSER GOAL: " + t.Goal
	request := RoleRequest{Role: roleAuditor, Directory: t.Project, OutDir: out, ProviderID: d.ProviderID, Model: d.Model, Effort: d.Effort, Prompt: prompt, ReadOnly: true}
	if err = e.attachRequest(&request, workAttachments(t)); err != nil {
		return err
	}
	if len(request.Attachments) > 0 {
		if err = e.saveJSON(t.ID, "inspection-"+d.ID+"-attachments.json", request.Attachments); err != nil {
			return err
		}
	}
	result, err := e.Runner.Run(ctx, request)
	if err != nil {
		return err
	}
	if err = e.validateAttachments(workAttachments(t)); err != nil {
		return err
	}
	report, sources, err := decodeInspectionReport(t.Project, result, workAttachments(t))
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = e.completeDispatch(t.ID, d, result); err != nil {
		return err
	}
	if err = e.saveJSON(t.ID, "inspection-sources.json", sources); err != nil {
		return err
	}
	if err = e.saveJSON(t.ID, "inspection-report.json", report); err != nil {
		return err
	}
	emitActivity(ctx, "message", "只读调查报告", report.Summary)
	_, err = e.Store.Update(t.ID, func(saved *Task) error {
		if saved.State == stateCancelling || saved.State == stateCancelled {
			return context.Canceled
		}
		saved.Audit = report
		saved.State = stateInspected
		return nil
	})
	return err
}

func decodeInspectionReport(project string, result *AgentResult, inputs ...[]Attachment) (*Audit, map[string]string, error) {
	if result == nil || result.Session == "" {
		return nil, nil, errors.New("inspection returned no identified report")
	}
	var report Audit
	if err := json.Unmarshal(result.Output, &report); err != nil {
		return nil, nil, err
	}
	if report.Summary == "" {
		return nil, nil, errors.New("inspection report is missing its summary")
	}
	var refs []Attachment
	if len(inputs) > 0 {
		refs = inputs[0]
	}
	attachments, err := inspectionAttachmentSources(refs, report.ReadAttachments)
	if err != nil {
		return nil, nil, err
	}
	if len(report.ReadFiles) == 0 && len(refs) > 0 && (len(attachments) > 0 || len(report.Unknowns) > 0) {
		return &report, attachments, nil
	}
	sources, err := inspectionSources(project, report.ReadFiles)
	if err != nil {
		return nil, nil, err
	}
	for id, hash := range attachments {
		sources[id] = hash
	}
	return &report, sources, nil
}

func (e *Engine) beginInspection(t *Task) (*Dispatch, error) {
	d := &Dispatch{ID: randomID(), Role: roleAuditor, Round: 0, Status: "running", Started: time.Now().UTC().Format(time.RFC3339Nano)}
	_, err := e.Store.Update(t.ID, func(saved *Task) error {
		if saved.State == stateCancelling || saved.State == stateCancelled {
			return context.Canceled
		}
		plan := saved.ModelPlan
		if plan == (ModelPlan{}) {
			plan = e.defaultPlan()
		}
		id, model, effort := plan.selection(roleAuditor, "")
		provider, model, effort, lookupErr := e.resolveSelection(id, model, effort)
		if lookupErr != nil {
			return lookupErr
		}
		d.ProviderID, d.ProviderName, d.Model, d.Effort = provider.ID, provider.Name, model, effort
		d.PlanRevision = saved.PlanRevision
		d.StartEvidence = "inspection-" + d.ID + "-start.json"
		d.EndEvidence = "inspection-" + d.ID + "-result.json"
		saved.Runs = append(saved.Runs, *d)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d, e.saveJSON(t.ID, d.StartEvidence, d)
}

func inspectionSources(project string, paths []string) (map[string]string, error) {
	if len(paths) == 0 || len(paths) > 100 {
		return nil, errors.New("inspection must cite between 1 and 100 source files")
	}
	result := map[string]string{}
	var total int64
	for _, path := range paths {
		if !safeRelative(path) {
			return nil, fmt.Errorf("invalid inspection source: %s", path)
		}
		resolved, err := filepath.EvalSymlinks(filepath.Join(project, filepath.FromSlash(path)))
		if err != nil || !pathInside(project, resolved) {
			return nil, errors.New("inspection source is unavailable or outside the project")
		}
		fingerprint, err := fileFingerprint(project, path, &total)
		if err != nil || fingerprint == deletedFingerprint {
			return nil, fmt.Errorf("inspection source cannot be verified: %s", path)
		}
		result[path] = fingerprint
	}
	return result, nil
}
