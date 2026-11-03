package reliable

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func parseHarnessModel(raw []byte) (ModelOption, bool) {
	if !harnessOptionalFieldsValid(raw) {
		return ModelOption{}, false
	}
	var row struct {
		ID           string                     `json:"id"`
		Name         string                     `json:"name"`
		Input        []string                   `json:"input_modalities"`
		Output       []string                   `json:"output_modalities"`
		Context      *float64                   `json:"context_window"`
		MaxOutput    *float64                   `json:"max_output_tokens"`
		Capabilities map[string]json.RawMessage `json:"api_capabilities"`
		Requirements map[string]json.RawMessage `json:"client_requirements"`
	}
	if json.Unmarshal(raw, &row) != nil || strings.TrimSpace(row.ID) == "" {
		return ModelOption{}, false
	}
	capability := bytes.TrimSpace(row.Capabilities["anthropic_messages"])
	if len(capability) == 0 || capability[0] != '{' {
		return ModelOption{}, false
	}
	if !catalogModalities(row.Input, []string{"text", "image"}) || !catalogModalities(row.Output, []string{"text"}) {
		return ModelOption{}, false
	}
	for _, number := range []*float64{row.Context, row.MaxOutput} {
		if number != nil && (*number <= 0 || *number > 9007199254740991 || *number != float64(int64(*number))) {
			return ModelOption{}, false
		}
	}
	if !harnessModelVersionCompatible(row.ID, row.Requirements["deepseek-harness"]) {
		return ModelOption{}, false
	}
	if row.Name == "" {
		row.Name = row.ID
	}
	return ModelOption{ID: row.ID, Name: row.Name, Efforts: []string{"off", "low", "high", "max"}, DefaultEffort: "high"}, true
}

func harnessOptionalFieldsValid(raw []byte) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return false
	}
	for _, key := range []string{"input_modalities", "output_modalities", "context_window", "max_output_tokens", "client_requirements"} {
		if value, exists := fields[key]; exists && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return false
		}
	}
	return true
}

func catalogModalities(values, allowed []string) bool {
	if values == nil {
		return true
	}
	for _, value := range values {
		for _, candidate := range allowed {
			if value == candidate {
				return true
			}
		}
	}
	return false
}

func harnessModelVersionCompatible(id string, raw json.RawMessage) bool {
	known := id == "deepseek-flash" || id == "deepseek-v4-pro"
	if len(raw) == 0 {
		return known
	}
	var requirement map[string]json.RawMessage
	if json.Unmarshal(raw, &requirement) != nil || requirement == nil {
		return false
	}
	minimum, exists := requirement["min_version"]
	if !exists {
		return known
	}
	var text string
	if json.Unmarshal(minimum, &text) != nil {
		return false
	}
	version, err := semver.StrictNewVersion(text)
	if err != nil {
		return false
	}
	current := semver.MustParse("0.2.1-alpha.5")
	return !current.LessThan(version)
}
