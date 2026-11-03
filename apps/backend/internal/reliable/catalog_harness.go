package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
)

func harnessCatalogToken(raw []byte) (string, error) {
	var doc struct {
		Version int `yaml:"version"`
		Records map[string]struct {
			Kind    string `yaml:"kind"`
			Payload struct {
				Version int    `yaml:"version"`
				Token   string `yaml:"token"`
				Issuer  string `yaml:"issuer"`
			} `yaml:"payload"`
		} `yaml:"records"`
	}
	if yaml.Unmarshal(raw, &doc) != nil {
		return "", errors.New("native account credential format unavailable")
	}
	record := doc.Records["deepseek-account-platform/default"]
	if doc.Version != 1 || record.Kind != "grant" || record.Payload.Version != 1 || record.Payload.Token == "" || record.Payload.Issuer != "https://platform.deepseek.com" {
		return "", errors.New("native account grant unavailable")
	}
	return record.Payload.Token, nil
}

func harnessModelCatalog(ctx context.Context, credentials []byte) ([]ModelOption, error) {
	token, err := harnessCatalogToken(credentials)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.deepseek.com/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-dsh-auth-token", token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "deepseek-harness/0.2.1-alpha.5 (+https://github.com/deepseek-ai/deepseek-harness)")
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("catalog redirects are disabled") }}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.New("native account model catalog request failed")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("native account model catalog is unavailable")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return nil, errors.New("native account model catalog exceeds its limit")
	}
	var listing struct {
		Data []json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &listing) != nil {
		return nil, errors.New("native account model catalog is malformed")
	}
	var models []ModelOption
	seen := map[string]bool{}
	for _, row := range listing.Data {
		option, ok := parseHarnessModel(row)
		if ok && !seen[option.ID] {
			seen[option.ID] = true
			models = append(models, option)
		}
	}
	if len(models) == 0 {
		return nil, errors.New("native account model catalog has no compatible models")
	}
	return models, nil
}
