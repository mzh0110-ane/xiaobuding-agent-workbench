package reliable

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const codexDiscoveryLimit = 256

var codexReleaseDirectory = regexp.MustCompile(`^[0-9a-fA-F]{12,64}$`)

type codexExecutableCandidate struct {
	path     string
	modified time.Time
}

func managedCodexRoot(configured string) (string, bool) {
	local := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(local) || !filepath.IsAbs(configured) {
		return "", false
	}
	root := filepath.Join(local, "OpenAI", "Codex", "bin")
	parent := filepath.Dir(filepath.Dir(filepath.Clean(configured)))
	if runtime.GOOS == "windows" {
		return root, strings.EqualFold(parent, root) && strings.EqualFold(filepath.Base(configured), "codex.exe")
	}
	return root, parent == root && filepath.Base(configured) == "codex.exe"
}

func resolveCodexExecutable(configured string) (string, error) {
	if root, managed := managedCodexRoot(configured); managed {
		return discoverManagedCodex(root)
	}
	if !filepath.IsAbs(configured) {
		return "", errors.New("codex_executable_invalid: configured Codex path must be absolute")
	}
	info, err := os.Stat(configured)
	if err != nil {
		return "", fmt.Errorf("codex_executable_unavailable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("codex_executable_invalid: configured Codex path is not a regular file")
	}
	return configured, nil
}

func discoverManagedCodex(root string) (string, error) {
	if err := validateCodexManagedRoot(root); err != nil {
		return "", fmt.Errorf("codex_executable_unavailable: %w", err)
	}
	directory, err := os.Open(root)
	if err != nil {
		return "", fmt.Errorf("codex_executable_unavailable: %w", err)
	}
	defer func() { _ = directory.Close() }()
	entries, err := directory.ReadDir(codexDiscoveryLimit + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("codex_executable_unavailable: %w", err)
	}
	if len(entries) > codexDiscoveryLimit {
		return "", errors.New("codex_discovery_limit: managed Codex directory has too many entries")
	}
	candidates := make([]codexExecutableCandidate, 0, len(entries))
	for _, entry := range entries {
		if candidate, ok := managedCodexCandidate(root, entry); ok {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return "", errors.New("codex_executable_unavailable: no complete native Codex build was found in the managed installation")
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modified.Equal(candidates[j].modified) {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].modified.After(candidates[j].modified)
	})
	return candidates[0].path, nil
}

func validateCodexManagedRoot(root string) error {
	for path := filepath.Clean(root); ; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() || codexFileRedirected(info) {
			return errors.New("managed Codex installation contains a redirected directory")
		}
		if parent := filepath.Dir(path); parent == path {
			return nil
		}
	}
}

func managedCodexCandidate(root string, entry os.DirEntry) (codexExecutableCandidate, bool) {
	if !codexReleaseDirectory.MatchString(entry.Name()) {
		return codexExecutableCandidate{}, false
	}
	dir, err := entry.Info()
	if err != nil || !dir.IsDir() || codexFileRedirected(dir) {
		return codexExecutableCandidate{}, false
	}
	path := filepath.Join(root, entry.Name(), "codex.exe")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || codexFileRedirected(info) || !codexNativeHeader(path, info) {
		return codexExecutableCandidate{}, false
	}
	return codexExecutableCandidate{path: path, modified: info.ModTime()}, true
}

func codexNativeHeader(path string, expected os.FileInfo) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !os.SameFile(expected, info) || info.Size() < 90 {
		return false
	}
	var dos [64]byte
	if _, err = file.ReadAt(dos[:], 0); err != nil || string(dos[:2]) != "MZ" {
		return false
	}
	offset := int64(binary.LittleEndian.Uint32(dos[60:64]))
	if offset < 64 || offset > 1<<20 || offset+26 > info.Size() {
		return false
	}
	var header [26]byte
	if _, err = file.ReadAt(header[:], offset); err != nil {
		return false
	}
	return validCodexPEHeader(header, offset, info.Size()) && validCodexPESections(file, header, offset, info.Size())
}

func validCodexPEHeader(header [26]byte, offset, size int64) bool {
	if string(header[:4]) != "PE\x00\x00" || binary.LittleEndian.Uint16(header[6:8]) == 0 {
		return false
	}
	machine := binary.LittleEndian.Uint16(header[4:6])
	if machine != 0x14c && machine != 0x8664 && machine != 0xaa64 {
		return false
	}
	flags := binary.LittleEndian.Uint16(header[22:24])
	optionalSize := int64(binary.LittleEndian.Uint16(header[20:22]))
	magic := binary.LittleEndian.Uint16(header[24:26])
	minimum := int64(240)
	if magic == 0x10b {
		minimum = 224
	} else if magic != 0x20b {
		return false
	}
	return flags&0x0002 != 0 && flags&0x2000 == 0 && optionalSize >= minimum && offset+24+optionalSize <= size
}

func validCodexPESections(file *os.File, header [26]byte, offset, size int64) bool {
	count := int64(binary.LittleEndian.Uint16(header[6:8]))
	if count < 1 || count > 96 {
		return false
	}
	table := offset + 24 + int64(binary.LittleEndian.Uint16(header[20:22]))
	tableEnd := table + count*40
	if tableEnd > size {
		return false
	}
	hasData := false
	for index := int64(0); index < count; index++ {
		var section [40]byte
		if _, err := file.ReadAt(section[:], table+index*40); err != nil {
			return false
		}
		length := int64(binary.LittleEndian.Uint32(section[16:20]))
		if length == 0 {
			continue
		}
		start := int64(binary.LittleEndian.Uint32(section[20:24]))
		if start < tableEnd || start+length > size {
			return false
		}
		hasData = true
	}
	return hasData
}
