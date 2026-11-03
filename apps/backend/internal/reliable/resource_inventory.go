package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const resourceLimit = 250000
const resourceMetadataLimit = 64 << 20
const resourceProfileName = "xiaobuding-resource-profile.json"
const resourceLibraryName = "resource-library.json"

var resourceExtensions = []string{"png", "jpg", "jpeg", "webp", "gif", "wmf", "emf", "tif", "tiff", "bmp", "ico", "pdf", "docx", "xlsx", "pptx", "mp3", "mp4", "wav", "zip", "epub", "gz"}

type resourceManifest struct {
	Base      string            `json:"base"`
	Format    string            `json:"format"`
	Source    map[string]string `json:"source"`
	Resources map[string]string `json:"resources"`
}

type resourceMarker struct {
	Manifest string `json:"manifest"`
	Digest   string `json:"digest"`
	Base     string `json:"base"`
}

func binaryResource(path string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	lower := strings.ToLower(path)
	if (strings.HasPrefix(lower, "data/") || strings.HasPrefix(lower, "artifacts/")) && (ext == "json" || ext == "jsonl") {
		return true
	}
	for _, candidate := range resourceExtensions {
		if ext == candidate {
			return true
		}
	}
	return false
}

func resourcePathSafe(path string) bool {
	if !safeRelative(path) || strings.ContainsAny(path, "\r\n\t") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if !resourceComponentSafe(part) {
			return false
		}
	}
	return true
}

func resourceComponentSafe(part string) bool {
	if strings.TrimRight(part, " .") != part || part == "." || strings.ContainsAny(part, "*?<>|\"") {
		return false
	}
	base := strings.ToLower(strings.SplitN(part, ".", 2)[0])
	switch base {
	case "con", "aux", "prn", "nul":
		return false
	}
	return len(base) != 4 || (!strings.HasPrefix(base, "com") && !strings.HasPrefix(base, "lpt")) || base[3] < '1' || base[3] > '9'
}

func resourceInventory(ctx context.Context, project string) (*resourceManifest, error) {
	raw, err := git(ctx, project, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	if bytes.Count(raw, []byte{0}) <= 30000 {
		return nil, nil
	}
	if len(raw) > resourceMetadataLimit || bytes.Count(raw, []byte{0}) > resourceLimit {
		return nil, errors.New("source_snapshot_too_large: resource inventory exceeds its limit")
	}
	m := &resourceManifest{Source: map[string]string{}, Resources: map[string]string{}}
	aliases := map[string]string{}
	for _, row := range bytes.Split(raw, []byte{0}) {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		if len(row) == 0 {
			continue
		}
		if err = addResourceIndexEntry(m, aliases, row); err != nil {
			return nil, err
		}
	}
	if len(m.Source) > 30000 || len(m.Resources) == 0 {
		return nil, errors.New("source_snapshot_too_large: more than 30000 mutable source files")
	}
	if err = completeResourceInventory(ctx, project, m); err != nil {
		return nil, err
	}
	if err = resourceAdmissionPolicy(ctx, project, m); err != nil {
		return nil, err
	}
	return m, nil
}

func resourceAdmissionPolicy(ctx context.Context, project string, m *resourceManifest) error {
	source := make(map[string]bool, len(m.Source))
	for path := range m.Source {
		source[path] = true
	}
	if err := localSourceBudget(project, source); err != nil {
		return fmt.Errorf("source_snapshot_too_large: mutable source: %w", err)
	}
	var paths strings.Builder
	for path := range m.Resources {
		paths.WriteString(path)
		paths.WriteByte(0)
	}
	raw, err := localGit(ctx, project, []byte(paths.String()), "check-attr", "-z", "--stdin", "filter", "working-tree-encoding")
	if err != nil {
		return err
	}
	fields := nulPaths(raw)
	if len(fields) != len(m.Resources)*6 {
		return errors.New("resource_inventory: incomplete resource attribute check")
	}
	for i := 0; i < len(fields); i += 3 {
		if fields[i+2] != "unspecified" && fields[i+2] != "unset" {
			return fmt.Errorf("resource_inventory: transformed resource is unsupported: %s", fields[i])
		}
	}
	return nil
}

func addResourceIndexEntry(m *resourceManifest, aliases map[string]string, row []byte) error {
	fields := bytes.SplitN(row, []byte{'\t'}, 2)
	if len(fields) != 2 {
		return errors.New("resource_inventory: malformed index")
	}
	meta := strings.Fields(string(fields[0]))
	path := string(fields[1])
	if len(meta) != 3 || meta[0] != "100644" || meta[2] != "0" || !resourcePathSafe(path) || locallyExcluded(path) {
		return fmt.Errorf("resource_inventory: unsupported source entry %q", path)
	}
	key := strings.ToLower(path)
	if old, ok := aliases[key]; ok && old != path {
		return errors.New("resource_inventory: filename case collision")
	}
	aliases[key] = path
	if binaryResource(path) {
		m.Resources[path] = meta[1]
	} else {
		m.Source[path] = meta[1]
	}
	return nil
}

func completeResourceInventory(ctx context.Context, project string, m *resourceManifest) error {
	head, err := git(ctx, project, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	m.Base = strings.TrimSpace(string(head))
	format, err := git(ctx, project, "rev-parse", "--show-object-format")
	if err != nil {
		return err
	}
	m.Format = strings.TrimSpace(string(format))
	raw, err := git(ctx, project, "ls-tree", "-r", "-z", m.Base)
	if err != nil {
		return err
	}
	if len(raw) > resourceMetadataLimit {
		return errors.New("resource_inventory: tree metadata exceeds limit")
	}
	if err = validateResourceTree(m, raw); err != nil {
		return err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if len(data) > resourceMetadataLimit {
		return errors.New("resource_inventory: manifest exceeds limit")
	}
	return nil
}

func validateResourceTree(m *resourceManifest, raw []byte) error {
	count := 0
	for _, row := range bytes.Split(raw, []byte{0}) {
		if len(row) == 0 {
			continue
		}
		count++
		parts := bytes.SplitN(row, []byte{'\t'}, 2)
		if len(parts) != 2 {
			return errors.New("resource_inventory: malformed tree")
		}
		meta := strings.Fields(string(parts[0]))
		path := string(parts[1])
		oid := m.Source[path]
		if oid == "" {
			oid = m.Resources[path]
		}
		if len(meta) != 3 || meta[0] != "100644" || meta[1] != "blob" || oid != meta[2] {
			return errors.New("resource_inventory: index differs from frozen tree")
		}
	}
	if count != len(m.Source)+len(m.Resources) {
		return errors.New("resource_inventory: incomplete tree")
	}
	return nil
}
