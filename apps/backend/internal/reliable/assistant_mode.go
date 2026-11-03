package reliable

import (
	"errors"
	"fmt"
)

const assistantCustom = "custom"

func (e *Engine) validateAssistantMode(plan ModelPlan, readOnly bool) error {
	switch plan.AssistantMode {
	case "":
		return nil
	case assistantCustom, connectionCodex, connectionHarness, connectionZcode:
	default:
		return errors.New("assistant_mode_invalid: select an available native assistant or custom collaboration")
	}
	if !readOnly && plan.Reviewer != connectionCodex {
		return errors.New("assistant_review_required: every assistant mode retains independent Codex review")
	}
	if plan.AssistantMode == assistantCustom {
		return nil
	}
	if plan.AssistantMode == connectionZcode {
		return e.validateZcodePreset(plan, readOnly)
	}
	if plan.Auditor != plan.AssistantMode {
		return errors.New("assistant_selection_mismatch: the selected assistant must perform project planning")
	}
	_, model, effort, err := e.resolveSelection(plan.Auditor, plan.AuditorModel, plan.AuditorEffort)
	if err != nil || readOnly {
		return err
	}
	return e.validateAssistantModeRoles(plan, model, effort)
}

func (e *Engine) validateAssistantModeRoles(plan ModelPlan, model, effort string) error {
	for _, role := range plan.assignmentKeys() {
		if role == roleReviewer || (role == roleAuditor && plan.AssistantMode == connectionZcode) {
			continue
		}
		id, choice, strength := plan.selection(role, "")
		if id != plan.AssistantMode {
			return fmt.Errorf("assistant_selection_mismatch: %s must use the selected native assistant", role)
		}
		_, resolved, actualEffort, lookupErr := e.resolveSelection(id, choice, strength)
		if lookupErr != nil {
			return lookupErr
		}
		if resolved != model || actualEffort != effort {
			return fmt.Errorf("assistant_selection_mismatch: %s model and reasoning must match the selected assistant; use custom collaboration for separate roles", role)
		}
	}
	return nil
}

func (e *Engine) validateZcodePreset(plan ModelPlan, readOnly bool) error {
	if readOnly {
		if plan.Auditor != connectionZcode {
			return errors.New("assistant_selection_mismatch: ZCode read-only mode must use its own native planner")
		}
		_, _, _, err := e.resolveSelection(plan.Auditor, plan.AuditorModel, plan.AuditorEffort)
		return err
	}
	if plan.Auditor != connectionCodex {
		return errors.New("assistant_zcode_audit_required: ZCode implements while Codex plans and independently reviews")
	}
	domain := ""
	if domains := plan.domains(); len(domains) > 0 {
		domain = domains[0]
	}
	id, model, effort := plan.selection(roleImplementer, domain)
	_, resolved, strength, err := e.resolveSelection(id, model, effort)
	if err != nil {
		return err
	}
	return e.validateAssistantModeRoles(plan, resolved, strength)
}
