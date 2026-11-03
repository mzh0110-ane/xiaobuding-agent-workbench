package reliable

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type HarnessRunner struct {
	Executable, Entry, Home, Provider, Model string
	Redact                                   func([]byte) []byte
	RedactorSource                           func() func([]byte) []byte
}

func (r *HarnessRunner) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	redact := snapshotRedactor(r.Redact, r.RedactorSource)
	if err := os.MkdirAll(req.OutDir, 0700); err != nil {
		return nil, err
	}
	imagesPatch, err := r.nativeImagePatch(req)
	if err != nil {
		return nil, err
	}
	patch, err := json.MarshalIndent(harnessPatch(req, r.Provider, r.Model), "", "  ")
	if err != nil {
		return nil, err
	}
	patchPath := filepath.Join(req.OutDir, "role-policy.yaml")
	if err = os.WriteFile(patchPath, patch, 0600); err != nil {
		return nil, err
	}
	// The account provider logs its model catalog with console.info. Keep that
	// diagnostic on stderr so stdout remains the strict native JSON protocol.
	preload := filepath.Join(req.OutDir, "native-diagnostics.cjs")
	if err = os.WriteFile(preload, []byte("const util = require('node:util');\nconsole.info = (...args) => process.stderr.write(util.format(...args) + '\\n');\n"), 0600); err != nil {
		return nil, err
	}
	env := isolatedEnvironment(os.Environ(), map[string]string{"DSH_HOME": r.Home, "PYTHONUTF8": "1", "PYTHONIOENCODING": "utf-8", "PYTHONDONTWRITEBYTECODE": "1"}, []string{"DEEPSEEK_API_KEY", "DEEPSEEK_BASE_URL", "DEEPSEEK_API_BASE", "DEEPSEEK_AUTH_TOKEN"})
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	inherited, cleanup, err := r.inheritedExtensions(ctx, req, env)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	args := []string{"--require", preload, r.Entry, "--profile", "headless"}
	if inherited != "" {
		args = append(args, "--patch", inherited)
	}
	if imagesPatch != "" {
		args = append(args, "--patch", imagesPatch)
	}
	args = append(args, "--patch", patchPath, "--json", "-")
	prompt := nativeAttachmentPrompt(req) + "\nReturn only one JSON object matching this schema, without Markdown fences or extra text:\n" + schemaForRequest(req)
	stdout, stderr, _, runErr := runProcess(observeNativeProcess(ctx, "harness", redact), req.Directory, env, prompt, r.Executable, args...)
	for name, raw := range map[string][]byte{"events.jsonl": stdout, "stderr.txt": stderr} {
		if err = os.WriteFile(filepath.Join(req.OutDir, name), redact(raw), 0600); err != nil {
			return nil, err
		}
	}
	if runErr != nil {
		return nil, fmt.Errorf("native Harness %s failed: %w; inspect its saved event log", req.Role, runErr)
	}
	result, err := parseHarnessEvents(stdout, req.Directory)
	if err != nil {
		return nil, err
	}
	result.Output = redact(result.Output)
	if err = os.WriteFile(filepath.Join(req.OutDir, "response.json"), result.Output, 0600); err != nil {
		return nil, err
	}
	return result, nil
}

func harnessPatch(req RoleRequest, provider, model string) []map[string]any {
	if req.Model != "" {
		model = req.Model
	}
	effort := "max"
	if req.Effort != "" {
		effort = req.Effort
	}
	modelConfig := map[string]any{"provider": provider, "model": model}
	if req.Effort != "" || req.WorkMode == "" {
		modelConfig["reasoningEffort"] = effort
	}
	mode := "danger-full-access"
	if req.ReadOnly {
		mode = "read-only"
	}
	preset := "xiaobuding-" + mode
	patch := []map[string]any{
		{"id": "credentials", "disabled": false}, {"id": "authorization", "disabled": false}, {"id": "deepseek-account", "disabled": false},
		{"id": "agent-default-model", "config": modelConfig},
		{"id": "sandbox-policy", "config": map[string]any{"mode": mode, "workspaceRoot": req.Directory}},
		{"id": "approval", "config": map[string]any{"policy": "never"}},
		{"id": "permission", "config": map[string]any{"presets": map[string]any{preset: map[string]string{"sandbox": mode, "approval": "never"}}, "defaultPreset": preset}},
	}
	for _, id := range []string{"tool-subagent", "tool-subagent-fork", "tool-workflow", "tool-working-directory"} {
		patch = append(patch, map[string]any{"id": id, "disabled": true})
	}
	return patch
}
