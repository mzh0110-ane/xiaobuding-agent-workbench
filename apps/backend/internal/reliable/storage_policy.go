package reliable

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const storageScanLimit = 250000

type StorageEntry struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Bytes        int64  `json:"bytes"`
	Files        int64  `json:"files"`
	SkippedLinks int64  `json:"skipped_links"`
}

type StorageUsage struct {
	Root       string         `json:"root"`
	TempRoot   string         `json:"temp_root"`
	ScannedAt  string         `json:"scanned_at"`
	Partial    bool           `json:"partial"`
	Entries    []StorageEntry `json:"entries"`
	TotalBytes int64          `json:"total_bytes"`
}

func (e *Engine) StorageUsage(ctx context.Context) (*StorageUsage, error) {
	root, err := openStorageRoot(e.Root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	report := &StorageUsage{Root: e.Root, TempRoot: filepath.Join(e.Root, "temp"), ScannedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	remaining := storageScanLimit
	for _, name := range []string{"evidence", "worktrees", "attachments", "context", "temp", "cache"} {
		entry := StorageEntry{Name: name, Path: filepath.Join(e.Root, name)}
		partial, scanErr := scanStorageEntry(ctx, root, name, &entry, &remaining)
		if scanErr != nil {
			return nil, scanErr
		}
		report.Partial = report.Partial || partial
		report.TotalBytes += entry.Bytes
		report.Entries = append(report.Entries, entry)
	}
	return report, nil
}

func openStorageRoot(path string) (*os.Root, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(filepath.Clean(absolute), filepath.Clean(resolved)) {
		return nil, errors.New("storage_boundary: application data root must not be a symbolic link")
	}
	return os.OpenRoot(absolute)
}

func scanStorageEntry(ctx context.Context, root *os.Root, name string, entry *StorageEntry, remaining *int) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return true, nil
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		entry.SkippedLinks++
		return true, nil
	}
	partial := false
	err = fs.WalkDir(root.FS(), name, func(path string, item fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if *remaining <= 0 {
			partial = true
			return fs.SkipAll
		}
		*remaining--
		if walkErr != nil {
			partial = true
			return nil
		}
		if item.Type()&os.ModeSymlink != 0 {
			entry.SkippedLinks++
			return nil
		}
		if item.Type().IsRegular() {
			fileInfo, statErr := item.Info()
			if statErr != nil {
				partial = true
				return nil
			}
			entry.Files++
			entry.Bytes += fileInfo.Size()
		}
		return nil
	})
	return partial, err
}
