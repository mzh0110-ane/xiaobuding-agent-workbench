package reliable

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TempCleanupReport struct {
	DryRun         bool  `json:"dry_run"`
	MinimumAgeDays int   `json:"minimum_age_days"`
	Files          int64 `json:"files"`
	Bytes          int64 `json:"bytes"`
	RemovedFiles   int64 `json:"removed_files"`
	RemovedBytes   int64 `json:"removed_bytes"`
	SkippedActive  int64 `json:"skipped_active"`
	Partial        bool  `json:"partial"`
}

func (e *Engine) CleanupTemporary(ctx context.Context, dryRun bool) (*TempCleanupReport, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !dryRun && len(e.active) > 0 {
		return nil, errors.New("storage_busy: temporary cleanup requires idle tasks")
	}
	storageRuntime.Lock()
	defer storageRuntime.Unlock()
	root, err := openStorageRoot(e.Root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	report := &TempCleanupReport{DryRun: dryRun, MinimumAgeDays: 7}
	info, statErr := root.Lstat("temp")
	if statErr == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
		return nil, errors.New("storage_boundary: temporary root must be an owned directory")
	}
	file, err := root.Open("temp")
	if errors.Is(err, os.ErrNotExist) {
		return report, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	entries, err := file.ReadDir(storageScanLimit)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	report.Partial = len(entries) == storageScanLimit
	for _, entry := range entries {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		if err = e.cleanTemporaryEntry(ctx, root, entry, report); err != nil {
			return nil, err
		}
	}
	return report, nil
}

func (e *Engine) cleanTemporaryEntry(ctx context.Context, root *os.Root, entry fs.DirEntry, report *TempCleanupReport) error {
	if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasPrefix(entry.Name(), "process-") {
		return nil
	}
	relative := filepath.Join("temp", entry.Name())
	if storageRuntime.active[filepath.Join(e.Root, relative)] {
		report.SkippedActive++
		return nil
	}
	markerInfo, err := root.Lstat(filepath.Join(relative, storageOwnerFile))
	if err != nil || !markerInfo.Mode().IsRegular() || markerInfo.Size() != int64(len(storageOwnerVersion)) {
		return nil
	}
	marker, err := root.ReadFile(filepath.Join(relative, storageOwnerFile))
	if err != nil || string(marker) != storageOwnerVersion {
		return nil
	}
	entryUsage, eligible, err := staleTemporaryUsage(ctx, root, relative, time.Now().Add(-7*24*time.Hour))
	if err != nil || !eligible {
		return err
	}
	report.Files += entryUsage.Files
	report.Bytes += entryUsage.Bytes
	if report.DryRun {
		return nil
	}
	if err = root.RemoveAll(relative); err != nil {
		return err
	}
	report.RemovedFiles += entryUsage.Files
	report.RemovedBytes += entryUsage.Bytes
	return nil
}

func staleTemporaryUsage(ctx context.Context, root *os.Root, relative string, cutoff time.Time) (StorageEntry, bool, error) {
	usage := StorageEntry{}
	eligible := true
	remaining := storageScanLimit
	err := fs.WalkDir(root.FS(), filepath.ToSlash(relative), func(path string, item fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		remaining--
		if walkErr != nil || remaining < 0 || item.Type()&os.ModeSymlink != 0 {
			eligible = false
			return fs.SkipAll
		}
		info, err := item.Info()
		if err != nil || info.ModTime().After(cutoff) {
			eligible = false
			return fs.SkipAll
		}
		if info.Mode().IsRegular() && item.Name() != storageOwnerFile {
			usage.Bytes += info.Size()
			usage.Files++
		}
		return nil
	})
	return usage, eligible, err
}
