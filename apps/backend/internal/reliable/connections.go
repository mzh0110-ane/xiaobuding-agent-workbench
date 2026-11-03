package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type ConnectionSettings struct {
	Models     []ModelOption `json:"models,omitempty"`
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Kind       string        `json:"kind"`
	Executable string        `json:"executable,omitempty"`
	Entry      string        `json:"entry,omitempty"`
	Home       string        `json:"home"`
	Provider   string        `json:"provider"`
	Model      string        `json:"model"`
}

type ProviderRouter struct{ Runners map[string]AgentRunner }

func (r *ProviderRouter) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	runner, ok := r.Runners[req.ProviderID]
	if !ok {
		return nil, errors.New("selected model connection has no configured runner")
	}
	return runner.Run(ctx, req)
}

func buildConnections(settings Settings) ([]Provider, *ProviderRouter, func([]byte) []byte, error) {
	providers := make([]Provider, 0, len(settings.Connections))
	router := &ProviderRouter{Runners: make(map[string]AgentRunner)}
	configs := make([][]byte, 0, len(settings.Connections))
	var nativeSecrets []string
	seen := make(map[string]bool)
	for _, c := range settings.Connections {
		if !filepath.IsAbs(c.Home) || c.Model == "" || seen[c.ID] {
			return nil, nil, nil, errors.New("model connections require unique IDs, absolute homes and explicit models")
		}
		seen[c.ID] = true
		provider, runner, config, err := discoverConnection(settings, c)
		if err != nil {
			return nil, nil, nil, err
		}
		providers = append(providers, provider)
		configs = append(configs, config)
		if c.Kind == connectionZcode {
			nativeSecrets = append(nativeSecrets, zcodeSecrets(config)...)
		}
		if c.Kind == "harness" {
			nativeSecrets = append(nativeSecrets, nativeCredentialSecrets(config)...)
		}
		router.Runners[c.ID] = runner
	}
	if len(providers) == 0 {
		providers = append(providers, Provider{ID: connectionHarness, Name: "DeepSeek Harness", ToolKind: "harness", Model: settings.Model, Detail: "请配置原生 Harness 或 Codex 的 ChatGPT 登录连接"})
	}
	redact := makeRedactor(bytes.Join(configs, []byte("\n")), nativeSecrets...)
	for _, runner := range router.Runners {
		switch typed := runner.(type) {
		case *CodexRunner:
			typed.Redact = redact
		case *ZcodeRunner:
			typed.Redact = redact
		case *HarnessRunner:
			typed.Redact = redact
		}
	}
	return providers, router, redact, nil
}

func discoverConnection(settings Settings, c ConnectionSettings) (Provider, AgentRunner, []byte, error) {
	p := Provider{ID: c.ID, Name: c.Name, ToolKind: c.Kind, Model: c.Model, Detail: "已发现本机配置，连接状态以实际执行为准"}
	switch c.Kind {
	case connectionZcode:
		return discoverZcode(c, p)
	case connectionCodex:
		return discoverCodex(settings, c, p)
	case "harness":
		if c.ID != connectionHarness || (c.Provider != "deepseek-official" && c.Provider != "deepseek-account") || !filepath.IsAbs(c.Entry) || !filepath.IsAbs(c.Executable) {
			return p, nil, nil, errors.New("DeepSeek connection requires the native Harness runtime and provider")
		}
		credentials, readErr := os.ReadFile(filepath.Join(c.Home, ".credentials.yaml"))
		p.Available = regularFile(c.Entry) && regularFile(c.Executable) && readErr == nil
		if !p.Available {
			p.Detail = "请检查原生 Harness 程序及其自己的登录或连接配置"
		}
		models, catalogErr := harnessModelCatalog(context.Background(), credentials)
		setModelCatalog(&p, models, "harness_account", catalogErr)
		return p, &HarnessRunner{Executable: c.Executable, Entry: c.Entry, Home: c.Home, Model: c.Model, Provider: c.Provider}, credentials, nil
	default:
		return p, nil, nil, errors.New("connection kind must be codex or harness")
	}
}

func discoverCodex(settings Settings, c ConnectionSettings, p Provider) (Provider, AgentRunner, []byte, error) {
	if c.ID != connectionCodex || c.Provider != "openai" {
		return p, nil, nil, errors.New("codex connection must use the official OpenAI provider")
	}
	config, err := os.ReadFile(filepath.Join(c.Home, "config.toml"))
	p.Available = err == nil && hasChatGPTLogin(c.Home)
	if !p.Available {
		p.Detail = "请先在本机通过 ChatGPT 登录 Codex"
	}
	models, catalogErr := codexModelCatalog(c.Home)
	setModelCatalog(&p, models, "codex_cache", catalogErr)
	return p, &CodexRunner{Executable: settings.CodexPath, Home: c.Home, Model: c.Model, Provider: "openai"}, config, nil
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func hasChatGPTLogin(home string) bool {
	raw, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		return false
	}
	var auth struct {
		Mode string `json:"auth_mode"`
	}
	return json.Unmarshal(raw, &auth) == nil && auth.Mode == "chatgpt"
}
