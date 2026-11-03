package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func recordMaterializedResources(ctx context.Context, dir string, available map[string]string) error {
	path, err := profileMarkerPath(ctx, dir)
	if err != nil {
		return err
	}
	path += ".available"
	previous := map[string]string{}
	raw, err := readResourceMetadata(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		if err = json.Unmarshal(raw, &previous); err != nil {
			return err
		}
	}
	for name, oid := range previous {
		if available[name] != oid {
			return fmt.Errorf("resource_changed: acknowledged input missing or changed: %s", name)
		}
	}
	if len(previous) == len(available) {
		return nil
	}
	return replaceResourceJSON(path, available)
}

func replaceResourceJSON(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".resource-state-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	_, writeErr := file.Write(raw)
	if err = errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func safeResourceDestination(dir, path string) (string, error) {
	if !resourcePathSafe(path) {
		return "", errors.New("resource_path: unsafe requested path")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", err
	}
	defer func() { _ = root.Close() }()
	parent := ""
	for _, part := range strings.Split(path, "/")[:len(strings.Split(path, "/"))-1] {
		parent = filepath.Join(parent, part)
		info, err := root.Lstat(parent)
		if os.IsNotExist(err) {
			if err = root.Mkdir(parent, 0700); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("resource_path: destination parent is not an ordinary directory")
		}
	}
	return filepath.Join(dir, filepath.FromSlash(path)), nil
}

func materializeResources(ctx context.Context, dir string, paths []string) error {
	if len(paths) == 0 || len(paths) > 64 {
		return errors.New("resource_request: request must contain 1 to 64 exact paths")
	}
	m, marker, err := loadResourceProfile(ctx, dir)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("resource_request: no frozen resource library")
	}
	if err = checkMaterializationBudget(ctx, dir, m, marker, paths); err != nil {
		return err
	}
	for _, path := range paths {
		oid, ok := m.Resources[path]
		if !ok {
			return fmt.Errorf("resource_request: input is not in the frozen library: %s", path)
		}
		if err = materializeResource(ctx, dir, path, oid, m.Format); err != nil {
			return err
		}
	}
	_, err = TakeSnapshot(ctx, dir)
	return err
}

func materializeResource(ctx context.Context, dir, path, oid, format string) error {
	target, err := safeResourceDestination(dir, path)
	if err != nil {
		return err
	}
	if _, err = os.Lstat(target); err == nil {
		var total int64
		got, err := resourceFingerprint(ctx, dir, path, format, &total)
		if err != nil {
			return err
		}
		if got != oid {
			return errors.New("resource_changed: existing input differs; preserved without overwrite")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := readResourceBlob(ctx, dir, oid)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = publishResourceCopy(ctx, dir, path, data); err != nil {
		return err
	}
	var total int64
	got, err := resourceFingerprint(ctx, dir, path, format, &total)
	if err != nil {
		return err
	}
	if got != oid {
		return errors.New("resource_changed: materialized bytes do not match the frozen object")
	}
	emitActivity(ctx, "phase", "已取出并校验输入资源", path)
	return nil
}

func publishResourceCopy(ctx context.Context, dir, path string, data []byte) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	temporary := filepath.Join(filepath.Dir(filepath.FromSlash(path)), ".input-resource-"+randomID())
	file, err := root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(temporary) }()
	_, writeErr := file.Write(data)
	if err = errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	// Both links refer only to the new task-local copy, never to original input.
	return root.Link(temporary, filepath.FromSlash(path))
}

func checkMaterializationBudget(ctx context.Context, dir string, m *resourceManifest, marker *resourceMarker, paths []string) error {
	snap, err := takeResourceSnapshot(ctx, dir, m, marker)
	if err != nil {
		return err
	}
	var used int64
	for path := range snap.AvailableResources {
		info, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		used += info.Size()
	}
	seen := map[string]bool{}
	for _, path := range paths {
		oid := m.Resources[path]
		if oid == "" {
			return fmt.Errorf("resource_request: unknown frozen input: %s", path)
		}
		if seen[path] || snap.AvailableResources[path] != "" {
			continue
		}
		seen[path] = true
		size, err := resourceBlobSize(ctx, dir, oid)
		if err != nil {
			return err
		}
		used += size
		if used > 2<<30 {
			return errors.New("resource_size: requested inputs exceed the 2 GiB task budget")
		}
	}
	return nil
}

func resourceBlobSize(ctx context.Context, dir, oid string) (int64, error) {
	raw, err := git(ctx, dir, "cat-file", "-s", oid)
	if err != nil {
		return 0, err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil || size < 0 || size > 64<<20 {
		return 0, errors.New("resource_size: requested input exceeds 64 MiB")
	}
	return size, nil
}

func readResourceBlob(ctx context.Context, dir, oid string) ([]byte, error) {
	size, err := resourceBlobSize(ctx, dir, oid)
	if err != nil {
		return nil, err
	}
	data, err := git(ctx, dir, "cat-file", "blob", oid)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != size {
		return nil, errors.New("resource_changed: object size mismatch")
	}
	return data, nil
}
