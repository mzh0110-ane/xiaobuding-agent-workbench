package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const storageOwnerFile = ".xiaobuding-temp-owner"
const storageOwnerVersion = "xiaobuding-runtime-temp-v1\n"

type storageRootKey struct{}

var storageRuntime = struct {
	sync.Mutex
	active map[string]bool
}{active: make(map[string]bool)}

func storageContext(ctx context.Context, root string) context.Context {
	return context.WithValue(ctx, storageRootKey{}, root)
}

func prepareStorageCommand(ctx context.Context, cmd *exec.Cmd) (func(), error) {
	rootPath, _ := ctx.Value(storageRootKey{}).(string)
	if rootPath == "" {
		return func() {}, nil
	}
	storageRuntime.Lock()
	defer storageRuntime.Unlock()
	root, err := openStorageRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	if err = root.MkdirAll("temp", 0700); err != nil {
		return nil, err
	}
	info, err := root.Lstat("temp")
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("storage_boundary: temporary root must be an owned directory")
	}
	name := "process-" + randomID()
	if err = root.Mkdir(filepath.Join("temp", name), 0700); err != nil {
		return nil, err
	}
	relative := filepath.Join("temp", name)
	if err = root.WriteFile(filepath.Join(relative, storageOwnerFile), []byte(storageOwnerVersion), 0600); err != nil {
		_ = root.RemoveAll(relative)
		return nil, err
	}
	path := filepath.Join(rootPath, relative)
	storageRuntime.active[path] = true
	applyStorageEnvironment(cmd, rootPath, path)
	return func() { releaseStorageCommand(rootPath, relative, path) }, nil
}

func applyStorageEnvironment(cmd *exec.Cmd, rootPath, scratch string) {
	base := cmd.Env
	if base == nil {
		base = os.Environ()
	}
	cache := filepath.Join(rootPath, "cache")
	overrides := map[string]string{
		"TEMP": scratch, "TMP": scratch, "TMPDIR": scratch, "GOTMPDIR": scratch,
		"npm_config_cache": filepath.Join(cache, "npm"), "PIP_CACHE_DIR": filepath.Join(cache, "pip"),
		"UV_CACHE_DIR": filepath.Join(cache, "uv"), "PLAYWRIGHT_BROWSERS_PATH": filepath.Join(cache, "ms-playwright"),
		"PYTHONDONTWRITEBYTECODE": "1",
		"GOCACHE":                 storageCacheValue(base, "GOCACHE", filepath.Join(cache, "go")),
		"GOMODCACHE":              storageCacheValue(base, "GOMODCACHE", filepath.Join(cache, "gomod")),
		"GOLANGCI_LINT_CACHE":     storageCacheValue(base, "GOLANGCI_LINT_CACHE", filepath.Join(cache, "golangci")),
	}
	cmd.Env = isolatedEnvironment(base, overrides, nil)
}

func storageCacheValue(base []string, name, fallback string) string {
	for _, item := range base {
		pair := strings.SplitN(item, "=", 2)
		if len(pair) != 2 || !strings.EqualFold(pair[0], name) {
			continue
		}
		if filepath.IsAbs(pair[1]) && !strings.EqualFold(filepath.VolumeName(pair[1]), "C:") {
			return pair[1]
		}
	}
	return fallback
}

func codexStorageArguments(ctx context.Context, args []string) ([]string, error) {
	rootPath, _ := ctx.Value(storageRootKey{}).(string)
	if rootPath == "" {
		return args, nil
	}
	root, err := openStorageRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	for _, item := range []struct{ key, path string }{{"log_dir", "log"}, {"sqlite_home", "state"}} {
		relative := filepath.Join("cache", "codex", item.path)
		if err = root.MkdirAll(relative, 0700); err != nil {
			return nil, err
		}
		encoded, _ := json.Marshal(filepath.Join(rootPath, relative))
		args = append(args, "-c", item.key+"="+string(encoded))
	}
	return append(args, "-c", `history.persistence="none"`), nil
}

func releaseStorageCommand(rootPath, relative, path string) {
	storageRuntime.Lock()
	defer storageRuntime.Unlock()
	defer delete(storageRuntime.active, path)
	root, err := openStorageRoot(rootPath)
	if err != nil {
		return
	}
	defer func() { _ = root.Close() }()
	_ = root.RemoveAll(relative)
}
