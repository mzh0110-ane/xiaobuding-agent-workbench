package reliable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const localGitDirectory = ".git"

const localExcludeRules = `
# Xiaobuding local recovery points: private and generated files stay local.
.env
.env.*
!.env.example
!.env.sample
!.env.template
auth.json
credentials.json
.netrc
.pypirc
.npmrc
.credentials*
*.key
*.pem
*.p12
*.pfx
id_rsa
id_ed25519
id_ecdsa
id_dsa
.codex/
codex-home/
.ssh/
.aws/
node_modules/
.venv/
venv/
__pycache__/
.pytest_cache/
.mypy_cache/
.next/
dist/
build/
coverage/
.cache/
*.pyc
`

func locallyExcluded(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		switch strings.ToLower(part) {
		case ".codex", "codex-home", ".ssh", ".aws", "node_modules", ".venv", "venv", "__pycache__", ".pytest_cache", ".mypy_cache", ".next", "dist", "build", "coverage", ".cache":
			return true
		}
	}
	name := strings.ToLower(parts[len(parts)-1])
	if name == ".env.example" || name == ".env.sample" || name == ".env.template" {
		return false
	}
	switch name {
	case ".env", "auth.json", "credentials.json", ".netrc", ".pypirc", ".npmrc", "id_rsa", "id_ed25519", "id_ecdsa", "id_dsa":
		return true
	}
	for _, suffix := range []string{".key", ".pem", ".p12", ".pfx", ".pyc"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return strings.HasPrefix(name, ".env.") || strings.HasPrefix(name, ".credentials")
}

func inspectLocalProjectTree(ctx context.Context, project string) error {
	visited := 0
	return filepath.WalkDir(project, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == project {
			return nil
		}
		rel, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		if rel == localGitDirectory && entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			return filepath.SkipDir
		}
		if locallyExcluded(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == localGitDirectory {
			return errors.New("local_snapshot_nested_git: nested repositories need a separate project folder")
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("local_snapshot_link: source links are not supported")
		}
		visited++
		if visited > 100000 {
			return errors.New("local_snapshot_size: choose a smaller project folder")
		}
		return nil
	})
}

func installLocalExcludes(project string) error {
	path := filepath.Join(project, localGitDirectory, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if bytes.Contains(raw, []byte("# Xiaobuding local recovery points:")) {
		return nil
	}
	return os.WriteFile(path, append(raw, []byte(localExcludeRules)...), 0600)
}

func localSourcePaths(ctx context.Context, project string, hasHead bool) ([]string, error) {
	trackedRaw, err := localGit(ctx, project, nil, "ls-files", "--cached", "-z")
	if err != nil {
		return nil, err
	}
	tracked := make(map[string]bool)
	for _, path := range nulPaths(trackedRaw) {
		tracked[path] = true
	}
	changed := make(map[string]bool)
	if hasHead {
		raw, diffErr := localGit(ctx, project, nil, "diff", "HEAD", "--name-only", "--no-renames", "-z", "--")
		if diffErr != nil {
			return nil, diffErr
		}
		for _, path := range nulPaths(raw) {
			changed[path] = true
		}
	}
	raw, err := localGit(ctx, project, nil, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	paths := make(map[string]bool)
	for _, path := range nulPaths(raw) {
		if !safeRelative(path) {
			return nil, errors.New("local_snapshot_path: a source filename is unsupported")
		}
		if locallyExcluded(path) && (!tracked[path] || changed[path]) {
			return nil, fmt.Errorf("local_snapshot_private: private or generated material conflicts with project ignore rules: %s", path)
		}
		paths[path] = true
	}
	if err = localSourceBudget(project, paths); err != nil {
		return nil, err
	}
	return localStagePaths(paths, tracked, changed, hasHead)
}

func localStagePaths(paths, tracked, changed map[string]bool, hasHead bool) ([]string, error) {
	stage := make(map[string]bool)
	for path := range paths {
		if (!tracked[path] || changed[path] || !hasHead) && !locallyExcluded(path) {
			stage[path] = true
		}
	}
	for path := range changed {
		if locallyExcluded(path) {
			return nil, errors.New("local_snapshot_private: existing private files require separate handling")
		}
		stage[path] = true
	}
	result := make([]string, 0, len(stage))
	for path := range stage {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

func localSourceBudget(project string, paths map[string]bool) error {
	if len(paths) > 30000 {
		return errors.New("local_snapshot_size: source exceeds 30000 files")
	}
	var total int64
	for path := range paths {
		info, err := os.Lstat(filepath.Join(project, filepath.FromSlash(path)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("local_snapshot_link: source links and submodules are not supported")
		}
		total += info.Size()
		if info.Size() > 16<<20 || total > 256<<20 {
			return errors.New("local_snapshot_size: source exceeds the supported size limit")
		}
	}
	return nil
}

func nulPaths(raw []byte) []string {
	var paths []string
	for _, value := range bytes.Split(raw, []byte{0}) {
		if len(value) > 0 {
			paths = append(paths, string(value))
		}
	}
	return paths
}
