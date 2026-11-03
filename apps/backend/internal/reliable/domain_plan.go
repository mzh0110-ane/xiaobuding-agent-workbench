package reliable

import (
	"errors"
	"strings"
)

const (
	connectionCodex   = "codex"
	connectionHarness = "deepseek"
	modeFullstack     = "fullstack"
	domainFrontend    = "frontend"
	domainBackend     = "backend"
)

type ModelOption struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Efforts       []string `json:"efforts"`
	DefaultEffort string   `json:"default_effort"`
}

func (p ModelPlan) domains() []string {
	switch p.Mode {
	case domainFrontend:
		return []string{domainFrontend}
	case domainBackend:
		return []string{domainBackend}
	case modeFullstack:
		return []string{domainFrontend, domainBackend}
	default:
		return nil
	}
}

func (p ModelPlan) assignmentKeys() []string {
	if p.Mode == "" {
		return []string{roleAuditor, roleImplementer, roleReviewer}
	}
	return append([]string{roleAuditor, roleReviewer}, p.domains()...)
}

func (p ModelPlan) selection(role, domain string) (string, string, string) {
	if role == roleImplementer && domain != "" {
		role = domain
	}
	switch role {
	case roleAuditor:
		return p.Auditor, p.AuditorModel, p.AuditorEffort
	case roleReviewer:
		return p.Reviewer, p.ReviewerModel, p.ReviewerEffort
	case domainFrontend:
		return p.Frontend, p.FrontendModel, p.FrontendEffort
	case domainBackend:
		return p.Backend, p.BackendModel, p.BackendEffort
	default:
		return p.Implementer, p.ImplementerModel, p.ImplementerEffort
	}
}

func (e *Engine) resolveSelection(id, model, effort string) (Provider, string, string, error) {
	p, err := e.provider(id)
	if err != nil {
		return p, "", "", err
	}
	if model == "" {
		model = p.Model
	}
	options := p.Models
	if len(options) == 0 {
		options = []ModelOption{{ID: p.Model}}
	}
	for _, option := range options {
		if option.ID != model {
			continue
		}
		if effort == "" {
			return p, model, option.DefaultEffort, nil
		}
		for _, allowed := range option.Efforts {
			if allowed == effort {
				return p, model, effort, nil
			}
		}
		return p, "", "", errors.New("selected reasoning effort is unsupported for this model")
	}
	return p, "", "", errors.New("selected model is not present in this tool's native catalog")
}

func (e *Engine) defaultTeamPlan() ModelPlan {
	plan := e.defaultPlan()
	codex := plan.Implementer
	if _, err := e.provider(connectionCodex); err == nil {
		codex = connectionCodex
	}
	plan.Mode, plan.Frontend, plan.Backend = modeFullstack, plan.Auditor, codex
	plan.Auditor, plan.Reviewer = codex, codex
	return plan
}

func splitDomainPaths(raw string) []string {
	var paths []string
	for _, value := range strings.Split(raw, ",") {
		if path := strings.TrimSpace(value); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func validateUserDomainPaths(plan ModelPlan) error {
	for _, domain := range plan.domains() {
		for _, path := range plan.userDomainPaths(domain) {
			if path != "." && !safeRelative(path) {
				return errors.New("domain paths must be safe project-relative paths")
			}
		}
	}
	return nil
}

func (p ModelPlan) userDomainPaths(domain string) []string {
	if domain == domainFrontend {
		return splitDomainPaths(p.FrontendPaths)
	}
	return splitDomainPaths(p.BackendPaths)
}
