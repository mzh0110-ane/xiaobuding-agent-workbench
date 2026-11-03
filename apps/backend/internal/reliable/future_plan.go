package reliable

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type FutureModelPlan struct {
	Plan     ModelPlan `json:"plan"`
	Revision int       `json:"revision"`
	Evidence string    `json:"evidence"`
}

func futureSettingsState(task *Task) bool {
	if acceptancePending(task) {
		return true
	}
	switch task.State {
	case stateInspected, stateAccepted, stateCancelled, stateExhausted, stateAwaitingAcceptance:
		return true
	case stateBlocked, stateInterrupted:
		return task.Round >= MaxRounds && !domainStageResumeEligible(task)
	default:
		return false
	}
}

func continuationPlan(task *Task) ModelPlan {
	if task.FuturePlan != nil {
		return task.FuturePlan.Plan
	}
	return task.ModelPlan
}

func (e *Engine) UpdateFutureModelPlan(id string, expected int, plan ModelPlan) (*Task, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closing {
		return nil, errors.New("application is shutting down")
	}
	return e.Store.Update(id, func(task *Task) error {
		if task.ArchivedAt != "" {
			return errors.New("task_archived: restore the task before changing future settings")
		}
		if task.SuccessorTaskID != "" {
			return fmt.Errorf("batch_successor_exists: update the linked batch %s instead", task.SuccessorTaskID)
		}
		if !futureSettingsState(task) {
			return errors.New("settings_stage_changed: reload the current model settings")
		}
		revision := 0
		if task.FuturePlan != nil {
			revision = task.FuturePlan.Revision
		}
		if expected != revision {
			return errors.New("model plan changed in another window; reload the latest plan")
		}
		if plan.Mode != task.ModelPlan.Mode || plan.FrontendPaths != task.ModelPlan.FrontendPaths || plan.BackendPaths != task.ModelPlan.BackendPaths {
			return errors.New("task work domains and authorized paths cannot change")
		}
		if plan.Mode != "" && plan.Reviewer != connectionCodex {
			return errors.New("domain workflow independent review must use Codex")
		}
		if err := e.validateTaskPlan(plan, task.ReadOnly); err != nil {
			return err
		}
		future := &FutureModelPlan{Plan: plan, Revision: revision + 1}
		if err := e.persistFuturePlan(task.ID, future); err != nil {
			return err
		}
		task.FuturePlan = future
		return nil
	})
}

func (e *Engine) persistFuturePlan(id string, plan *FutureModelPlan) error {
	dir := filepath.Join(e.Root, "future-models", id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	plan.Evidence = filepath.Join(dir, fmt.Sprintf("plan-%d-%s.json", plan.Revision, randomID()))
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(plan.Evidence, raw, 0600)
}

func (e *Engine) promoteFuturePlan(task *Task) error {
	if task.FuturePlan == nil {
		return nil
	}
	plan := task.FuturePlan.Plan
	if err := e.validateTaskPlan(plan, task.ReadOnly); err != nil {
		return err
	}
	next := task.PlanRevision + 1
	name := fmt.Sprintf("model-plan-%d-%s.json", next, randomID())
	if err := e.saveJSON(task.ID, name, plan); err != nil {
		return err
	}
	task.ModelPlan, task.PlanRevision = plan, next
	task.PlanEvidence = append(task.PlanEvidence, name)
	task.FuturePlan = nil
	return nil
}

func (h *HTTP) updateFutureModels(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		ExpectedRevision int       `json:"expected_revision"`
		Plan             ModelPlan `json:"plan"`
	}
	if err := requestJSON(r, &body); err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	task, err := h.Engine.UpdateFutureModelPlan(id, body.ExpectedRevision, body.Plan)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, task)
}
