package reliable

import (
	"bytes"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Settings struct {
	Connections     []ConnectionSettings `json:"connections,omitempty"`
	DataRoot        string               `json:"data_root"`
	CodexPath       string               `json:"codex_path"`
	CodexHome       string               `json:"codex_home"`
	Model           string               `json:"model"`
	DemoProject     string               `json:"demo_project"`
	TeamDemoProject string               `json:"team_demo_project,omitempty"`
}

func randomID() string { return strings.ToLower(rand.Text()) }

func FromHome(home string) (*HTTP, error) {
	raw, err := os.ReadFile(filepath.Join(home, "reliable-settings.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var settings Settings
	if err = decodeStrict(raw, &settings); err != nil {
		return nil, err
	}
	for _, path := range []string{settings.DataRoot, settings.CodexHome, settings.CodexPath} {
		if !filepath.IsAbs(path) {
			return nil, errors.New("reliable workflow settings require absolute paths")
		}
	}
	if _, err = resolveCodexExecutable(settings.CodexPath); err != nil {
		if _, managed := managedCodexRoot(settings.CodexPath); !managed {
			return nil, err
		}
	}
	providers, runner, redact, err := buildConnections(settings)
	if err != nil {
		return nil, err
	}
	applyNativeModelHistory(settings, providers)
	store, err := OpenStore(filepath.Join(settings.DataRoot, "state.db"))
	if err != nil {
		return nil, err
	}
	if err = store.Recover(); err != nil {
		_ = store.Close()
		return nil, err
	}
	filters := &redactorSource{filter: redact}
	filters.bind(runner)
	engine := &Engine{Store: store, Root: settings.DataRoot, Runner: runner, Providers: providers, Redact: filters.apply}
	return &HTTP{Engine: engine, Token: rand.Text(), Model: settings.Model, DemoProject: settings.DemoProject, TeamDemoProject: settings.TeamDemoProject, PickProject: pickProject, ReloadCatalog: func() ([]Provider, error) {
		providers, _, nextFilter, err := buildConnections(settings)
		if err == nil {
			applyNativeModelHistory(settings, providers)
			filters.replace(nextFilter)
		}
		return providers, err
	}}, nil
}

func makeRedactor(config []byte, extra ...string) func([]byte) []byte {
	secrets := append(nativeCredentialSecrets(config), extra...)
	pattern := regexp.MustCompile(`(?i)(?:experimental_bearer_token|api_key|access_token|password)\s*=\s*["']([^"']+)["']`)
	for _, match := range pattern.FindAllSubmatch(config, -1) {
		if len(match[1]) >= 8 {
			secrets = append(secrets, string(match[1]))
		}
	}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(parts[0])
		if (strings.Contains(key, "API_KEY") || strings.Contains(key, "TOKEN") || strings.Contains(key, "SECRET")) && len(parts[1]) >= 8 {
			secrets = append(secrets, parts[1])
		}
	}
	seen := make(map[string]bool)
	filtered := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		if len(secret) >= 8 && !seen[secret] {
			filtered = append(filtered, secret)
			seen[secret] = true
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return len(filtered[i]) > len(filtered[j]) })
	return func(raw []byte) []byte {
		out := bytes.Clone(raw)
		for _, secret := range filtered {
			out = bytes.ReplaceAll(out, []byte(secret), []byte("[REDACTED]"))
		}
		return out
	}
}
