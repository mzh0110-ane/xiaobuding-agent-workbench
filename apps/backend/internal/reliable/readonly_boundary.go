package reliable

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func selectedProjectBoundary(project, root string, readOnly bool) error {
	if readOnly {
		return readOnlyProjectBoundary(project)
	}
	return localProjectBoundary(project, root)
}

func readOnlyProjectBoundary(project string) error {
	info, err := os.Stat(project)
	if err != nil || !info.IsDir() {
		return errors.New("local_snapshot_directory: choose an existing folder")
	}
	if filepath.Dir(project) == project {
		return errors.New("local_snapshot_boundary: choose a folder rather than an entire drive")
	}
	for _, name := range []string{".codex", ".zcode", "codex-home", ".ssh", ".git"} {
		if strings.EqualFold(filepath.Base(project), name) {
			return errors.New("local_snapshot_boundary: choose source or documentation rather than an account directory")
		}
	}
	return nil
}
