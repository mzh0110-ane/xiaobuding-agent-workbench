package reliable

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

func codexModelCatalog(home string) ([]ModelOption, error) {
	raw, err := os.ReadFile(filepath.Join(home, "models_cache.json"))
	if err != nil {
		return nil, err
	}
	if len(raw) > 8<<20 {
		return nil, errors.New("model catalog exceeds its size limit")
	}
	var cache struct {
		Models []struct {
			ID         string `json:"slug"`
			Name       string `json:"display_name"`
			Visibility string `json:"visibility"`
			Priority   int    `json:"priority"`
			Default    string `json:"default_reasoning_level"`
			Efforts    []struct {
				Effort string `json:"effort"`
			} `json:"supported_reasoning_levels"`
		} `json:"models"`
	}
	if err = json.Unmarshal(raw, &cache); err != nil {
		return nil, err
	}
	sort.SliceStable(cache.Models, func(i, j int) bool { return cache.Models[i].Priority < cache.Models[j].Priority })
	models := make([]ModelOption, 0, len(cache.Models))
	seen := map[string]bool{}
	for _, m := range cache.Models {
		if m.Visibility != "list" || m.ID == "" || seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		option := ModelOption{ID: m.ID, Name: m.Name, DefaultEffort: m.Default}
		if option.Name == "" {
			option.Name = option.ID
		}
		for _, e := range m.Efforts {
			if e.Effort != "" {
				option.Efforts = append(option.Efforts, e.Effort)
			}
		}
		models = append(models, option)
	}
	if len(models) == 0 {
		return nil, errors.New("native model catalog contains no visible models")
	}
	return models, nil
}

func setModelCatalog(p *Provider, models []ModelOption, source string, err error) {
	p.CatalogError = err != nil
	p.CatalogSource = source
	if err != nil {
		p.Models = []ModelOption{{ID: p.Model, Name: p.Model, Efforts: nil}}
		p.CatalogStatus = "目录未刷新，仅保留已配置模型；请在原工具确认后刷新"
		return
	}
	p.Models = models
	p.CatalogStatus = "已读取原生模型目录，实际可用性以执行结果为准"
	found := false
	for _, m := range models {
		if m.ID == p.Model {
			found = true
		}
	}
	if !found {
		if p.Model == "" {
			p.Model = models[0].ID
			return
		}
		p.Models = append([]ModelOption{{ID: p.Model, Name: p.Model + "（已配置，未验证）"}}, models...)
		p.CatalogStatus = "已读取原生目录；已配置模型未列出，保留原选择，实际可用性以执行结果为准"
	}
}
