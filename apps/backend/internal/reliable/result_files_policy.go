package reliable

import (
	"errors"
	"os"
	"path"
	"strings"
)

func resultComponentAllowed(component string) bool {
	if strings.HasPrefix(component, ".") || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
		return false
	}
	switch component {
	case "node_modules", "codex-home", "venv", "__pycache__", "agents.md", "claude.md", "skill.md",
		"auth.json", "credentials.json", "secrets.json", "secrets.yaml", "secrets.yml",
		"id_rsa", "id_ed25519", "id_ecdsa", "id_dsa":
		return false
	}
	switch path.Ext(component) {
	case ".key", ".pem", ".p12", ".pfx", ".db", ".sqlite", ".sqlite3", ".log":
		return false
	}
	return !strings.HasSuffix(component, ".instructions.md")
}

func openResultFile(root *os.Root, name string, limit int64) (*os.File, error) {
	if !resultPathAllowed(name) {
		return nil, errors.New("result_file_unsafe: private or invalid candidate path")
	}
	parts := strings.Split(name, "/")
	var finalInfo os.FileInfo
	for i := range parts {
		info, err := root.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return nil, errors.New("result_file_unsafe: candidate path is missing or redirected")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, errors.New("result_file_unsafe: candidate parent is not a directory")
		}
		finalInfo = info
	}
	if finalInfo == nil || !finalInfo.Mode().IsRegular() || finalInfo.Size() > limit {
		return nil, errors.New("result_file_limit: candidate must be a regular file within the read limit")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(finalInfo, opened) {
		_ = file.Close()
		return nil, errors.New("result_file_changed: candidate identity changed while opening")
	}
	return file, nil
}
