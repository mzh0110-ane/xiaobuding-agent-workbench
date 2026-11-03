package reliable

import (
	"encoding/json"
	"errors"
	"fmt"
)

func domainAuditInstructions(plan ModelPlan) string {
	if plan.Mode == "" {
		return ""
	}
	raw, _ := json.Marshal(map[string]any{"mode": plan.Mode, "frontend_user_paths": plan.userDomainPaths(domainFrontend), "backend_user_paths": plan.userDomainPaths(domainBackend)})
	return "\nDOMAIN CONTRACT: " + string(raw) + ". Return frontend_paths and backend_paths. Each enabled domain needs concrete nonempty paths within allowed_paths and any supplied user domain paths. Disabled domains return an empty array. Both-domain scopes must not overlap, including parent/child prefixes; assign shared interface/schema files to one writer only. Do not infer that all src code is frontend. Plan frontend then backend against the same source, with explicit API expectations."
}

func auditDomainPaths(a *Audit, domain string) []string {
	if domain == domainFrontend {
		return a.FrontendPaths
	}
	return a.BackendPaths
}

func validateDomainAudit(plan ModelPlan, audit *Audit) error {
	if err := normalizeAuditScopes(audit); err != nil {
		return err
	}
	for _, domain := range plan.domains() {
		paths := auditDomainPaths(audit, domain)
		if len(paths) == 0 {
			return fmt.Errorf("audit did not establish %s scope", domain)
		}
		for _, path := range paths {
			if path != "." && !safeRelative(path) {
				return errors.New("audit proposed an unsafe domain scope")
			}
			if !withinScope(path, audit.AllowedPaths) {
				return errors.New("domain scope exceeds the audited scope")
			}
			if user := plan.userDomainPaths(domain); len(user) > 0 && !withinScope(path, user) {
				return errors.New("domain scope exceeds the user domain authorization")
			}
		}
	}
	if plan.Mode == modeFullstack {
		for _, front := range audit.FrontendPaths {
			for _, back := range audit.BackendPaths {
				if withinScope(front, []string{back}) || withinScope(back, []string{front}) {
					return errors.New("frontend and backend scopes overlap; assign shared files to one writer")
				}
			}
		}
	}
	return nil
}
