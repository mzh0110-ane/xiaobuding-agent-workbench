package reliable

import (
	"context"
	"crypto/sha1" // Git SHA-1 repositories require the canonical Git blob identity.
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type resourceScan struct {
	ctx                        context.Context
	dir                        string
	manifest                   *resourceManifest
	snapshot                   *Snapshot
	sourcePaths                map[string]bool
	sourceBytes, resourceBytes int64
	visited                    int
}

func takeResourceSnapshot(ctx context.Context, dir string, m *resourceManifest, marker *resourceMarker) (*Snapshot, error) {
	head, err := git(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(head)) != m.Base {
		return nil, errors.New("resource_inventory: candidate history changed")
	}
	raw, err := git(ctx, dir, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	snap := &Snapshot{Head: m.Base, Files: map[string]string{}, ResourceHash: marker.Digest, AvailableResources: map[string]string{}}
	scan := &resourceScan{ctx: ctx, dir: dir, manifest: m, snapshot: snap, sourcePaths: map[string]bool{}}
	for _, path := range nulPaths(raw) {
		if m.Resources[path] == "" {
			scan.sourcePaths[path] = true
		}
	}
	for path := range m.Source {
		snap.Files[path] = deletedFingerprint
		scan.sourcePaths[path] = true
	}
	if len(scan.sourcePaths) > 30000 {
		return nil, errors.New("source snapshot exceeds 30000 mutable files")
	}
	if err = filepath.WalkDir(dir, scan.visit); err != nil {
		return nil, err
	}
	if err = validateResourceIndex(ctx, dir, m); err != nil {
		return nil, err
	}
	if err = recordMaterializedResources(ctx, dir, snap.AvailableResources); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(snap.Files))
	for path := range snap.Files {
		names = append(names, path)
	}
	sort.Strings(names)
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "resource-v1\n%s\n%s\n", snap.Head, snap.ResourceHash)
	for _, path := range names {
		_, _ = fmt.Fprintf(h, "%d:%s:%s\n", len(path), path, snap.Files[path])
	}
	snap.Hash = hex.EncodeToString(h.Sum(nil))
	return snap, nil
}

func (s *resourceScan) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if path == s.dir {
		return nil
	}
	rel, err := filepath.Rel(s.dir, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if rel == localGitDirectory {
		if entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	s.visited++
	if s.visited > 500000 {
		return errors.New("resource_inventory: materialized tree exceeds limit")
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return errors.New("resource_inventory: source links are unsupported")
	}
	if entry.IsDir() {
		if locallyExcluded(rel) {
			return filepath.SkipDir
		}
		return nil
	}
	if !resourcePathSafe(rel) {
		return errors.New("resource_inventory: unsafe physical filename")
	}
	if oid, ok := s.manifest.Resources[rel]; ok {
		return s.verifyResource(rel, oid)
	}
	if !s.sourcePaths[rel] {
		return nil
	}
	fingerprint, err := fileFingerprint(s.dir, rel, &s.sourceBytes)
	if err != nil {
		return err
	}
	s.snapshot.Files[rel] = fingerprint
	return nil
}

func (s *resourceScan) verifyResource(path, oid string) error {
	got, err := resourceFingerprint(s.ctx, s.dir, path, s.manifest.Format, &s.resourceBytes)
	if err != nil {
		return err
	}
	if got != oid {
		return fmt.Errorf("resource_changed: frozen input changed: %s", path)
	}
	s.snapshot.AvailableResources[path] = oid
	return nil
}

func resourceFingerprint(ctx context.Context, dir, path, format string, total *int64) (string, error) {
	info, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(path)))
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("resource_changed: input is not a regular file")
	}
	*total += info.Size()
	if info.Size() > 64<<20 || *total > 2<<30 {
		return "", errors.New("resource_size: input exceeds materialization budget")
	}
	var h hash.Hash
	if format == "sha1" {
		h = sha1.New()
	} else {
		h = sha256.New()
	}
	_, _ = fmt.Fprintf(h, "blob %d\x00", info.Size())
	file, err := os.Open(filepath.Join(dir, filepath.FromSlash(path)))
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	buffer := make([]byte, 64<<10)
	for {
		if err = ctx.Err(); err != nil {
			return "", err
		}
		n, readErr := file.Read(buffer)
		if n > 0 {
			_, _ = h.Write(buffer[:n])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validateResourceIndex(ctx context.Context, dir string, m *resourceManifest) error {
	raw, err := git(ctx, dir, "diff", "--cached", "--name-only", "--no-renames", "-z", m.Base, "--")
	if err != nil {
		return err
	}
	for _, path := range nulPaths(raw) {
		if _, ok := m.Resources[path]; ok {
			return fmt.Errorf("resource_changed: frozen resource staged: %s", path)
		}
	}
	return nil
}
