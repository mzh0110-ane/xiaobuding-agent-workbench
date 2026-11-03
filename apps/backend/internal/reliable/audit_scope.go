package reliable

import (
	"fmt"
	"strings"
)

func normalizeAuditScopes(audit *Audit) error {
	normalized := *audit
	for _, scope := range []struct {
		name  string
		paths *[]string
	}{
		{"allowed_paths", &normalized.AllowedPaths},
		{"frontend_paths", &normalized.FrontendPaths},
		{"backend_paths", &normalized.BackendPaths},
	} {
		paths, err := normalizeAuditScopePaths(*scope.paths, scope.name)
		if err != nil {
			return err
		}
		*scope.paths = paths
	}
	*audit = normalized
	return nil
}

func normalizeAuditScopePaths(paths []string, field string) ([]string, error) {
	if paths == nil {
		return nil, nil
	}
	result := make([]string, len(paths))
	for index, path := range paths {
		canonical := strings.TrimSuffix(path, "/")
		if canonical != "." && !safeRelative(canonical) {
			return nil, fmt.Errorf("audit proposed an unsafe scope: %s[%d]=%q", field, index, path)
		}
		result[index] = canonical
	}
	return result, nil
}
