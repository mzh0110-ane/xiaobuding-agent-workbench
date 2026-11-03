package reliable

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func discoverZcode(c ConnectionSettings, p Provider) (Provider, AgentRunner, []byte, error) {
	if c.ID != connectionZcode || !filepath.IsAbs(c.Entry) || !filepath.IsAbs(c.Executable) || c.Provider == "" {
		return p, nil, nil, errors.New("ZCode requires an explicit native entry, executable and provider")
	}
	raw, err := os.ReadFile(filepath.Join(c.Home, "v2", "provider_config.json"))
	p.Available = err == nil && regularFile(c.Entry) && regularFile(c.Executable)
	p.Models = c.Models
	p.CatalogSource = "zcode_native"
	p.CatalogStatus = "复用 ZCode 原生模型、配置和 skill；执行时核对实际模型。"
	return p, &ZcodeRunner{Executable: c.Executable, Entry: c.Entry, Home: c.Home, Provider: c.Provider, Model: c.Model}, raw, nil
}

func zcodeSecrets(raw []byte) []string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	var result []string
	var visit func(any)
	visit = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for key, item := range x {
				lower := strings.ToLower(key)
				if s, ok := item.(string); ok && len(s) >= 8 && (strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret")) {
					result = append(result, s)
				} else {
					visit(item)
				}
			}
		case []any:
			for _, item := range x {
				visit(item)
			}
		}
	}
	visit(value)
	return result
}
