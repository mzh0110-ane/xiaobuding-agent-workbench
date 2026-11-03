package reliable

import (
	"errors"
	"fmt"
)

type ModelPlan struct {
	AssistantMode     string `json:"assistant_mode,omitempty"`
	Auditor           string `json:"auditor"`
	Implementer       string `json:"implementer"`
	Reviewer          string `json:"reviewer"`
	Mode              string `json:"mode,omitempty"`
	Frontend          string `json:"frontend,omitempty"`
	Backend           string `json:"backend,omitempty"`
	FrontendPaths     string `json:"frontend_paths,omitempty"`
	BackendPaths      string `json:"backend_paths,omitempty"`
	AuditorModel      string `json:"auditor_model,omitempty"`
	ImplementerModel  string `json:"implementer_model,omitempty"`
	ReviewerModel     string `json:"reviewer_model,omitempty"`
	FrontendModel     string `json:"frontend_model,omitempty"`
	BackendModel      string `json:"backend_model,omitempty"`
	AuditorEffort     string `json:"auditor_effort,omitempty"`
	ImplementerEffort string `json:"implementer_effort,omitempty"`
	ReviewerEffort    string `json:"reviewer_effort,omitempty"`
	FrontendEffort    string `json:"frontend_effort,omitempty"`
	BackendEffort     string `json:"backend_effort,omitempty"`
}

type Provider struct {
	ToolKind      string        `json:"tool_kind"`
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Model         string        `json:"model"`
	Available     bool          `json:"available"`
	Detail        string        `json:"detail"`
	Models        []ModelOption `json:"models,omitempty"`
	CatalogSource string        `json:"catalog_source,omitempty"`
	CatalogStatus string        `json:"catalog_status,omitempty"`
	CatalogError  bool          `json:"catalog_error,omitempty"`
}

func (e *Engine) providerList() []Provider {
	e.catalogMu.RLock()
	defer e.catalogMu.RUnlock()
	if len(e.Providers) > 0 {
		return append([]Provider(nil), e.Providers...)
	}
	return []Provider{{ID: connectionHarness, Name: "DeepSeek", Model: "existing", Available: true, Detail: "已发现现有配置"}}
}

func (e *Engine) provider(id string) (Provider, error) {
	for _, p := range e.providerList() {
		if p.ID == id && p.Available {
			return p, nil
		}
	}
	return Provider{}, errors.New("selected model connection is unavailable")
}

func (e *Engine) defaultPlan() ModelPlan {
	id := connectionHarness
	if _, err := e.provider(id); err != nil {
		for _, p := range e.providerList() {
			if p.Available {
				id = p.ID
				break
			}
		}
	}
	plan := ModelPlan{Auditor: id, Implementer: id, Reviewer: id}
	if _, err := e.provider(connectionCodex); err == nil {
		plan.Implementer = connectionCodex
	}
	return plan
}

func (e *Engine) validatePlan(plan ModelPlan) error {
	if plan.Reviewer == connectionZcode {
		return errors.New("independent review must remain on the existing review provider")
	}
	if err := e.validateAssistantMode(plan, false); err != nil {
		return err
	}
	if plan.Mode != "" && plan.Reviewer != connectionCodex {
		return errors.New("domain workflow independent review must use Codex")
	}
	if plan.Mode != "" && plan.Mode != domainFrontend && plan.Mode != domainBackend && plan.Mode != modeFullstack {
		return errors.New("work mode must be frontend, backend or fullstack")
	}
	for _, role := range plan.assignmentKeys() {
		id, model, effort := plan.selection(role, "")
		if _, _, _, err := e.resolveSelection(id, model, effort); err != nil {
			return fmt.Errorf("%s: %w", role, err)
		}
	}
	return validateUserDomainPaths(plan)
}

func (e *Engine) UpdateModelPlan(id string, expected int, plan ModelPlan) (*Task, error) {
	task, err := e.Store.Get(id)
	if err != nil {
		return nil, err
	}
	if err := e.validateTaskPlan(plan, task.ReadOnly); err != nil {
		return nil, err
	}
	return e.Store.Update(id, func(t *Task) error {
		if acceptancePending(t) {
			return errors.New("local_checkpoint_pending: only future batch settings may change while acceptance completes")
		}
		if t.ArchivedAt != "" {
			return errors.New("task_archived: restore the task before editing its assignments")
		}
		if t.SuccessorTaskID != "" {
			return fmt.Errorf("batch_successor_exists: update the linked batch %s instead", t.SuccessorTaskID)
		}
		if t.PlanRevision != expected {
			return errors.New("model plan changed in another window; reload the latest plan")
		}
		if (t.PlanRevision > 0 || t.Worktree != "" || len(t.Runs) > 0) && (plan.Mode != t.ModelPlan.Mode || plan.FrontendPaths != t.ModelPlan.FrontendPaths || plan.BackendPaths != t.ModelPlan.BackendPaths) {
			return errors.New("task work domains and authorized paths cannot change; only future tools, models and efforts are editable")
		}
		switch t.State {
		case stateAccepted, stateCancelled, stateCancelling, stateApplying, stateExhausted, stateAwaitingAcceptance, stateInspected:
			return errors.New("this task has no editable future stages")
		}
		next := t.PlanRevision + 1
		name := fmt.Sprintf("model-plan-%d-%s.json", next, randomID())
		if err := e.saveJSON(id, name, plan); err != nil {
			return err
		}
		t.ModelPlan = plan
		t.PlanRevision = next
		t.PlanEvidence = append(t.PlanEvidence, name)
		return nil
	})
}

func (e *Engine) validateTaskPlan(plan ModelPlan, readOnly bool) error {
	if !readOnly {
		return e.validatePlan(plan)
	}
	if err := e.validateAssistantMode(plan, true); err != nil {
		return err
	}
	id, model, effort := plan.selection(roleAuditor, "")
	_, _, _, err := e.resolveSelection(id, model, effort)
	return err
}
