package reliable

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"path/filepath"
)

//go:embed harness-native.mjs
var harnessNativeBootstrap []byte

func (r *HarnessRunner) inheritedExtensions(ctx context.Context, req RoleRequest, env []string) (string, func(), error) {
	noop := func() {}
	manifest := filepath.Join(r.Home, "profiles", "desktop", "package.json")
	if _, err := os.Stat(manifest); os.IsNotExist(err) {
		return "", noop, nil
	}
	private, err := os.MkdirTemp("", "xiaobuding-native-extensions-")
	if err != nil {
		return "", noop, err
	}
	cleanup := func() { _ = os.RemoveAll(private) }
	bootstrap := filepath.Join(req.OutDir, "native-inheritance.mjs")
	if err = os.WriteFile(bootstrap, harnessNativeBootstrap, 0600); err != nil {
		cleanup()
		return "", noop, err
	}
	patch := filepath.Join(private, "inherited-extensions.yaml")
	inventory := filepath.Join(req.OutDir, "native-extensions.json")
	_, _, _, err = runProcess(ctx, req.Directory, env, "", r.Executable, bootstrap, r.Entry, r.Home, patch, inventory)
	if err != nil {
		cleanup()
		return "", noop, errors.New("native Harness extensions could not be prepared; check its existing desktop installation")
	}
	return patch, cleanup, nil
}
