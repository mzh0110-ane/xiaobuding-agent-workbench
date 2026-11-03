package reliable

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const eventTypeError = "error"

type RoleRequest struct {
	ResourceMode                                                                 bool
	Role, Directory, Prompt, OutDir, ProviderID, Domain, Model, Effort, WorkMode string
	ReadOnly                                                                     bool
	Attachments                                                                  []AttachmentInput `json:"attachments,omitempty"`
}
type AgentResult struct {
	Session string
	Output  []byte
}
type AgentRunner interface {
	Run(context.Context, RoleRequest) (*AgentResult, error)
}

func parseSession(events []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(events))
	scanner.Buffer(make([]byte, 4096), 8<<20)
	id := ""
	completed := false
	for scanner.Scan() {
		var event struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
			Message  string `json:"message"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return "", errors.New("malformed Codex event stream")
		}
		switch event.Type {
		case "thread.started":
			if id != "" || event.ThreadID == "" {
				return "", errors.New("ambiguous Codex session identity")
			}
			id = event.ThreadID
		case "turn.started":
			completed = false
		case "turn.completed":
			completed = true
		case eventTypeError:
			if !isCodexReconnectNotice(event.Message) {
				return "", errors.New("codex reported a failed turn; inspect the saved event log")
			}
			completed = false
		case "turn.failed":
			return "", errors.New("codex reported a failed turn; inspect the saved event log")
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if id == "" || !completed {
		return "", errors.New("codex did not finish an identified session")
	}
	return id, nil
}

func discoverChecks(dir string) []string {
	out := discoverNodeChecks(dir)
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		out = append(out, "go test ./...")
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "tests", "test*.py")); len(matches) > 0 {
		out = append(out, "python -m unittest discover -s tests -v")
	}
	return out
}

func discoverNodeChecks(dir string) []string {
	out := make([]string, 0)
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return out
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return out
	}
	for _, name := range []string{"test", "typecheck", "build"} {
		script := strings.TrimSpace(pkg.Scripts[name])
		if script != "" && !strings.HasPrefix(script, "echo ") && !strings.Contains(script, "no test specified") {
			manager := "npm run "
			if _, err := os.Stat(filepath.Join(dir, "pnpm-lock.yaml")); err == nil {
				manager = "pnpm run "
			}
			out = append(out, manager+name)
		}
	}
	return out
}

type limitedBuffer struct {
	buffer  bytes.Buffer
	cancel  context.CancelFunc
	observe func([]byte)
}

func (b *limitedBuffer) Bytes() []byte { return b.buffer.Bytes() }
func (b *limitedBuffer) Len() int      { return b.buffer.Len() }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 16<<20 {
		b.cancel()
		return 0, errors.New("process output exceeded 16 MiB")
	}
	n, err := b.buffer.Write(p)
	if b.observe != nil {
		b.observe(p[:n])
	}
	return n, err
}

func runProcess(ctx context.Context, dir string, env []string, input string, name string, args ...string) ([]byte, []byte, int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(input)
	cmd.WaitDelay = 2 * time.Second
	stdout := &limitedBuffer{cancel: cancel}
	stderr := &limitedBuffer{cancel: cancel}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	flush := observeProcessOutput(ctx, stdout, stderr)
	defer flush()
	err := runOwnedCommand(ctx, cmd)
	code := -1
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	return stdout.Bytes(), stderr.Bytes(), code, err
}

func runOwnedCommand(ctx context.Context, cmd *exec.Cmd) (runErr error) {
	storageCleanup, err := prepareStorageCommand(ctx, cmd)
	if err != nil {
		return err
	}
	storageReleasable := true
	defer func() {
		if storageReleasable {
			storageCleanup()
		}
	}()
	cmd.WaitDelay = 2 * time.Second
	prepareOwnedCommand(cmd)
	finish := trackOwnedCleanup(ctx)
	if err := cmd.Start(); err != nil {
		finish(nil)
		return err
	}
	jobCleanup, err := isolateProcess(cmd)
	if err != nil {
		killErr := cmd.Process.Kill()
		waitErr := cmd.Wait()
		finish(err)
		return errors.Join(err, killErr, waitErr, ctx.Err())
	}
	cleanup := sync.OnceValue(jobCleanup)
	storageReleasable = false
	stopCleanup := context.AfterFunc(ctx, func() { _ = cleanup() })
	defer func() {
		stopCleanup()
		cleanupErr := cleanup()
		storageReleasable = cleanupErr == nil
		proofErr := cleanupErr
		if proofErr == nil && !ownedProcessTreeAttested() {
			proofErr = errors.New("owned process tree cleanup cannot be attested on this platform")
		}
		finish(proofErr)
		if cleanupErr != nil {
			cleanupErr = fmt.Errorf("owned_process_cleanup: %w", cleanupErr)
		}
		runErr = errors.Join(runErr, ctx.Err(), cleanupErr)
	}()
	return cmd.Wait()
}

type CodexRunner struct {
	Executable, Home, Model string
	Provider                string
	Redact                  func([]byte) []byte
	RedactorSource          func() func([]byte) []byte
}

func (r *CodexRunner) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	executable, err := r.executableForContext(ctx)
	if err != nil {
		return nil, err
	}
	redact := snapshotRedactor(r.Redact, r.RedactorSource)
	if err := validateNativeImageInputs(req.Attachments); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(req.OutDir, 0700); err != nil {
		return nil, err
	}
	schema := schemaForRequest(req)
	schemaPath := filepath.Join(req.OutDir, "schema.json")
	if err := os.WriteFile(schemaPath, []byte(schema), 0600); err != nil {
		return nil, err
	}
	responsePath := filepath.Join(req.OutDir, "response.json")
	args, err := codexStorageArguments(ctx, r.arguments(req, schemaPath, responsePath))
	if err != nil {
		return nil, err
	}
	removed := []string{}
	if r.Provider == "openai" {
		removed = append(removed, "OPENAI_API_KEY")
	}
	env := isolatedEnvironment(os.Environ(), map[string]string{"CODEX_HOME": r.Home, "PYTHONUTF8": "1", "PYTHONIOENCODING": "utf-8", "PYTHONDONTWRITEBYTECODE": "1"}, removed)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	stdout, stderr, _, runErr := runProcess(observeNativeProcess(ctx, "codex", redact), req.Directory, env, nativeAttachmentPrompt(req), executable, args...)
	if err := os.WriteFile(filepath.Join(req.OutDir, "events.jsonl"), redact(stdout), 0600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(req.OutDir, "stderr.txt"), redact(stderr), 0600); err != nil {
		return nil, err
	}
	if runErr != nil {
		return nil, fmt.Errorf("codex %s failed: %w. Inspect its saved event log", req.Role, runErr)
	}
	id, err := parseSession(stdout)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(responsePath)
	if err != nil {
		return nil, err
	}
	if len(raw) > 1<<20 {
		return nil, errors.New("agent report exceeds 1 MiB")
	}
	if err := os.WriteFile(responsePath, redact(raw), 0600); err != nil {
		return nil, err
	}
	return &AgentResult{Session: id, Output: redact(raw)}, nil
}

func (r *CodexRunner) arguments(req RoleRequest, schemaPath, responsePath string) []string {
	args := []string{"exec", "--ephemeral", "-C", req.Directory, "-s", "danger-full-access", "-c", `approval_policy="never"`, "--json", "--output-schema", schemaPath, "-o", responsePath, "-"}
	if req.ReadOnly {
		args[5] = "read-only"
		args = append(args, "--skip-git-repo-check")
	}
	model := r.Model
	if req.Model != "" {
		model = req.Model
	}
	if model != "" {
		args = append(args, "-m", model)
	}
	if req.Effort != "" {
		args = append(args, "-c", `model_reasoning_effort="`+req.Effort+`"`)
	}
	if r.Provider != "" {
		args = append(args, "-c", `model_provider="`+r.Provider+`"`)
	}
	images, _ := nativeImageInputs(req.Attachments)
	for _, image := range images {
		args = append(args, "--image", image.Path)
	}
	return args
}

func runCheck(ctx context.Context, dir, command, logPath string, redact func([]byte) []byte) (Check, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	env := make([]string, 0)
	for _, v := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(v, "=", 2)[0])
		if !strings.Contains(key, "TOKEN") && !strings.Contains(key, "SECRET") && !strings.Contains(key, "API_KEY") {
			env = append(env, v)
		}
	}
	env = append(env, "PYTHONDONTWRITEBYTECODE=1", "PYTHONUTF8=1", "PYTHONIOENCODING=utf-8")
	var stdout, stderr []byte
	var code int
	var err error
	if runtime.GOOS == "windows" {
		stdout, stderr, code, err = runProcess(ctx, dir, env, "", "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", "$ErrorActionPreference='Stop'; [Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false); $OutputEncoding=[Console]::OutputEncoding; "+command+"; if (-not $?) { exit 1 }; if ($null -ne $LASTEXITCODE) { exit $LASTEXITCODE }")
	} else {
		stdout, stderr, code, err = runProcess(ctx, dir, env, "", "sh", "-c", command)
	}
	log := append(append([]byte("COMMAND: "+command+"\nSTDOUT:\n"), stdout...), append([]byte("\nSTDERR:\n"), stderr...)...)
	if redact != nil {
		log = redact(log)
	}
	if saveErr := os.WriteFile(logPath, log, 0600); saveErr != nil {
		return Check{}, saveErr
	}
	if err != nil && code == 0 {
		code = -1
	}
	return Check{Command: command, ExitCode: code, Log: filepath.Base(logPath)}, nil
}
