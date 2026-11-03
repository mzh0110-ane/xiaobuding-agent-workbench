package reliable

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type nativeModelObservation struct {
	ID         string      `json:"id"`
	Home       string      `json:"home"`
	Provider   string      `json:"provider"`
	Model      ModelOption `json:"model"`
	ObservedAt time.Time   `json:"observed_at"`
}

type nativeModelHistory struct {
	Version int                      `json:"version"`
	Entries []nativeModelObservation `json:"entries"`
}

func applyNativeModelHistory(settings Settings, providers []Provider) {
	path := filepath.Join(settings.DataRoot, "native-model-metadata.json")
	history := nativeModelHistory{Version: 1}
	if raw, err := os.ReadFile(path); err == nil && len(raw) <= 256<<10 {
		if json.Unmarshal(raw, &history) != nil || history.Version != 1 || len(history.Entries) > 64 {
			history = nativeModelHistory{Version: 1}
		}
	}
	changed := false
	for i := range providers {
		for _, connection := range settings.Connections {
			if connection.Kind != connectionCodex || connection.ID != providers[i].ID {
				continue
			}
			changed = applyModelObservation(&providers[i], connection, &history) || changed
		}
	}
	if changed {
		writeNativeModelHistory(path, history)
	}
}

func applyModelObservation(provider *Provider, connection ConnectionSettings, history *nativeModelHistory) bool {
	for i, model := range provider.Models {
		if model.ID != provider.Model {
			continue
		}
		if !provider.CatalogError && model.Name != provider.Model+"（已配置，未验证）" {
			entry := nativeModelObservation{ID: connection.ID, Home: connection.Home, Provider: connection.Provider, Model: model, ObservedAt: time.Now().UTC()}
			for j := range history.Entries {
				if history.Entries[j].ID == connection.ID {
					history.Entries[j] = entry
					return true
				}
			}
			history.Entries = append(history.Entries, entry)
			return true
		}
		for _, entry := range history.Entries {
			if entry.ID != connection.ID || filepath.Clean(entry.Home) != filepath.Clean(connection.Home) || entry.Provider != connection.Provider || entry.Model.ID != provider.Model || entry.ObservedAt.IsZero() {
				continue
			}
			provider.Models[i] = entry.Model
			provider.Models[i].Name += "（原生历史记录）"
			provider.CatalogSource = "codex_cache_history"
			provider.CatalogStatus = "沿用已读取的原生历史记录；当前目录未确认该模型，实际可用性以执行结果为准"
			return false
		}
	}
	return false
}

func writeNativeModelHistory(path string, history nativeModelHistory) {
	raw, err := json.MarshalIndent(history, "", "  ")
	if err != nil || os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".native-model-*")
	if err != nil {
		return
	}
	defer func() { _ = os.Remove(file.Name()) }()
	_, writeErr := file.Write(raw)
	closeErr := file.Close()
	if writeErr == nil && closeErr == nil {
		_ = os.Rename(file.Name(), path)
	}
}
