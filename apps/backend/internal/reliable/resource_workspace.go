package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func exclusiveJSON(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(raw)
	return errors.Join(writeErr, file.Sync(), file.Close())
}

func profileMarkerPath(ctx context.Context, dir string) (string, error) {
	raw, err := git(ctx, dir, "rev-parse", "--path-format=absolute", "--git-path", resourceProfileName)
	return strings.TrimSpace(string(raw)), err
}

func loadResourceProfile(ctx context.Context, dir string) (*resourceManifest, *resourceMarker, error) {
	path, err := profileMarkerPath(ctx, dir)
	if err != nil {
		return nil, nil, err
	}
	raw, err := readResourceMetadata(path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var marker resourceMarker
	if err = json.Unmarshal(raw, &marker); err != nil {
		return nil, nil, err
	}
	raw, err = readResourceMetadata(marker.Manifest)
	if err != nil {
		return nil, nil, err
	}
	if digest(raw) != marker.Digest {
		return nil, nil, errors.New("resource_inventory: frozen manifest changed")
	}
	var m resourceManifest
	if err = json.Unmarshal(raw, &m); err != nil {
		return nil, nil, err
	}
	if m.Base != marker.Base || (m.Format != "sha1" && m.Format != "sha256") {
		return nil, nil, errors.New("resource_inventory: invalid frozen baseline")
	}
	return &m, &marker, nil
}

func readResourceMetadata(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > resourceMetadataLimit {
		return nil, errors.New("resource_inventory: unsafe metadata")
	}
	return os.ReadFile(path)
}

func prepareResourceWorktree(ctx context.Context, root, project, id, dir, base string, m *resourceManifest) (string, string, error) {
	evidence := filepath.Join(root, "evidence", id)
	if err := os.MkdirAll(evidence, 0700); err != nil {
		return "", "", err
	}
	marker, err := prepareResourceManifest(evidence, base, m)
	if err != nil {
		return "", "", err
	}
	intent := filepath.Join(evidence, "resource-preparation.json")
	if err := writeResourcePathIndex(evidence, m); err != nil {
		return "", "", err
	}
	if _, err := os.Stat(dir); err == nil {
		return recoverResourceWorktree(ctx, project, dir, id, base, marker)
	} else if !os.IsNotExist(err) {
		return "", "", err
	}
	if _, err := os.Stat(intent); os.IsNotExist(err) {
		if err = exclusiveJSON(intent, marker); err != nil {
			return "", "", err
		}
	} else if err != nil {
		return "", "", err
	}
	emitActivity(ctx, "phase", "正在准备精简工作副本", fmt.Sprintf("源码 %d 个；题库等输入资源 %d 个按需取出。全部操作都在本机。", len(m.Source), len(m.Resources)))
	if _, err := git(ctx, project, "worktree", "add", "--no-checkout", "-b", "xiaobuding/"+id, dir, base); err != nil {
		return "", "", err
	}
	markerPath, err := profileMarkerPath(ctx, dir)
	if err != nil {
		return "", "", err
	}
	if err = exclusiveJSON(markerPath, marker); err != nil {
		return "", "", err
	}
	if err = initializeResourceWorktree(ctx, dir); err != nil {
		return "", "", err
	}
	return dir, base, nil
}

func prepareResourceManifest(evidence, base string, m *resourceManifest) (*resourceMarker, error) {
	target := filepath.Join(evidence, resourceLibraryName)
	raw, _ := json.Marshal(m)
	marker := &resourceMarker{Manifest: target, Digest: digest(raw), Base: base}
	if old, err := os.ReadFile(target); err == nil {
		if digest(old) != marker.Digest {
			return nil, errors.New("resource_inventory: preparation baseline changed")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	} else if err = exclusiveJSON(target, m); err != nil {
		return nil, err
	}
	return marker, nil
}

func initializeResourceWorktree(ctx context.Context, dir string) error {
	patterns := "/*\n"
	for _, ext := range resourceExtensions {
		patterns += "!*."
		for _, char := range ext {
			patterns += "[" + string(char) + strings.ToUpper(string(char)) + "]"
		}
		patterns += "\n"
	}
	patterns += "!/[dD][aA][tT][aA]/**/*.[jJ][sS][oO][nN]\n!/[dD][aA][tT][aA]/**/*.[jJ][sS][oO][nN][lL]\n!/[aA][rR][tT][iI][fF][aA][cC][tT][sS]/**/*.[jJ][sS][oO][nN]\n!/[aA][rR][tT][iI][fF][aA][cC][tT][sS]/**/*.[jJ][sS][oO][nN][lL]\n"
	if _, err := localGit(ctx, dir, []byte(patterns), "sparse-checkout", "set", "--no-cone", "--stdin"); err != nil {
		return err
	}
	_, err := git(ctx, dir, "read-tree", "-mu", "HEAD")
	if err != nil {
		return err
	}
	path, err := profileMarkerPath(ctx, dir)
	if err != nil {
		return err
	}
	return exclusiveJSON(path+".ready", map[string]bool{"ready": true})
}

func recoverResourceWorktree(ctx context.Context, project, dir, id, base string, want *resourceMarker) (string, string, error) {
	_, marker, err := loadResourceProfile(ctx, dir)
	if err != nil || marker == nil || *marker != *want {
		return "", "", errors.New("resource_inventory: incomplete candidate requires inspection; no files were reset")
	}
	path, err := profileMarkerPath(ctx, dir)
	if err != nil {
		return "", "", err
	}
	if _, err = os.Stat(path + ".ready"); err != nil {
		if err = recoverInitialWorktree(ctx, project, dir, id, base); err != nil {
			return "", "", errors.New("resource_inventory: interrupted sparse initialization requires inspection; candidate preserved")
		}
		if _, err = TakeSnapshot(ctx, dir); err != nil {
			return "", "", err
		}
		if err = exclusiveJSON(path+".ready", map[string]bool{"ready": true}); err != nil {
			return "", "", err
		}
	}
	if err = recoverInitialWorktree(ctx, project, dir, id, base); err != nil {
		return "", "", err
	}
	return dir, base, nil
}

func writeResourcePathIndex(evidence string, m *resourceManifest) error {
	paths := make([]string, 0, len(m.Resources))
	for path := range m.Resources {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	data := []byte(strings.Join(paths, "\n") + "\n")
	target := filepath.Join(evidence, "resource-paths.txt")
	if old, err := os.ReadFile(target); err == nil {
		if digest(old) != digest(data) {
			return errors.New("resource_inventory: resource path index changed")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	return errors.Join(writeErr, file.Sync(), file.Close())
}
