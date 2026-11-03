package reliable

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

//go:embed zcode-client.cjs
var zcodeClient []byte

//go:embed zcode-entry.cjs
var zcodeEntry []byte

//go:embed zcode-readonly.cjs
var zcodeReadOnly []byte

const connectionZcode = "zcode"
const zcodeDefaultEffort = "high"

type ZcodeRunner struct {
	Executable, Entry, Home, Provider, Model string
	Redact                                   func([]byte) []byte
}

func (r *ZcodeRunner) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	if err := os.MkdirAll(req.OutDir, 0700); err != nil {
		return nil, err
	}
	for name, raw := range map[string][]byte{"zcode-client.cjs": zcodeClient, "zcode-entry.cjs": zcodeEntry, "zcode-readonly.cjs": zcodeReadOnly} {
		if err := os.WriteFile(filepath.Join(req.OutDir, name), raw, 0600); err != nil {
			return nil, err
		}
	}
	effort := req.Effort
	if effort == "" {
		effort = zcodeDefaultEffort
	}
	model := req.Model
	if model == "" {
		model = r.Model
	}
	prompt := zcodePrompt(req)
	input, _ := json.Marshal(map[string]any{"entry": r.Entry, "node": r.Executable, "home": r.Home, "provider": r.Provider, "model": model, "effort": effort, "directory": req.Directory, "readOnly": req.ReadOnly, "prompt": prompt})
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	stdout, stderr, _, err := runProcess(observeNativeProcess(ctx, "codex", r.Redact), req.Directory, os.Environ(), string(input), r.Executable, filepath.Join(req.OutDir, "zcode-entry.cjs"))
	for name, raw := range map[string][]byte{"events.jsonl": stdout, "stderr.txt": stderr} {
		if r.Redact != nil {
			raw = r.Redact(raw)
		}
		if saveErr := os.WriteFile(filepath.Join(req.OutDir, name), raw, 0600); saveErr != nil {
			return nil, saveErr
		}
	}
	if err != nil {
		return nil, errors.New("ZCode native execution failed; inspect saved event log")
	}
	result, err := parseZcodeResult(stdout, req.ReadOnly)
	if err != nil {
		return nil, err
	}
	if r.Redact != nil {
		result.Output = r.Redact(result.Output)
	}
	if err = os.WriteFile(filepath.Join(req.OutDir, "response.json"), result.Output, 0600); err != nil {
		return nil, err
	}
	return result, nil
}

type zcodeStreamState struct {
	completed, readonlySession string
	result                     *AgentResult
}

func parseZcodeResult(raw []byte, readOnly ...bool) (*AgentResult, error) {
	scan := bufio.NewScanner(bytes.NewReader(raw))
	scan.Buffer(make([]byte, 4096), 8<<20)
	state := &zcodeStreamState{}
	for scan.Scan() {
		if err := state.consume(scan.Bytes()); err != nil {
			return nil, err
		}
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	if state.result == nil {
		return nil, errors.New("ZCode did not finish an identified session")
	}
	if len(readOnly) > 0 && readOnly[0] && state.readonlySession != state.result.Session {
		return nil, errors.New("ZCode read-only execution has no matching native permission proof")
	}
	return state.result, nil
}

func (state *zcodeStreamState) consume(raw []byte) error {
	var row struct {
		Type, Session, Response string
		Event                   struct {
			Type, SessionID, Session string
			PlanEnabled              bool
			Payload                  struct{ ResultType string }
		}
	}
	if json.Unmarshal(raw, &row) != nil {
		return errors.New("malformed ZCode stream")
	}
	switch row.Type {
	case "zcode.event":
		switch row.Event.Type {
		case "native_readonly_confirmed":
			if row.Event.PlanEnabled {
				state.readonlySession = row.Event.Session
			}
		case "turn.failed":
			return errors.New("ZCode native turn failed")
		case "turn.completed":
			if row.Event.Payload.ResultType == "success" {
				state.completed = row.Event.SessionID
			}
		}
	case "result":
		if state.result != nil || row.Session == "" || row.Session != state.completed || row.Response == "" {
			return errors.New("invalid ZCode native completion")
		}
		state.result = &AgentResult{Session: row.Session, Output: []byte(row.Response)}
	}
	return nil
}

func zcodePrompt(req RoleRequest) string {
	prompt := req.Prompt + "\nReturn only one JSON object matching this schema, without Markdown fences. Return instance data, not the schema itself. Source paths in read_files must be relative to the selected project directory:\n" + schemaForRequest(req)
	if len(req.Attachments) > 0 {
		prompt += "\nAttachments are file references, not proof of understanding. Inspect them with native tools or existing skills; explicitly report unreadable content.\n"
		for _, a := range req.Attachments {
			prompt += "\n" + a.Path
		}
	}
	return prompt
}
