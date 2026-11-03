package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type previewProcess struct {
	done chan struct{}
	err  error
}

func detectPreview(task *Task, snapshot *Snapshot) PreviewStatus {
	static := detectStaticPreview(task, snapshot)
	status := static
	root, err := os.OpenRoot(task.Worktree)
	if err != nil {
		return status
	}
	defer func() { _ = root.Close() }()
	file, err := openResultFile(root, "package.json", 64<<10)
	if err != nil {
		return status
	}
	raw, err := io.ReadAll(io.LimitReader(file, (64<<10)+1))
	_ = file.Close()
	var manifest struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err != nil || json.Unmarshal(raw, &manifest) != nil {
		return status
	}
	script := strings.TrimSpace(manifest.Scripts["dev"])
	if script == "" || !strings.HasPrefix(script, previewKindVite) {
		return status
	}
	status.Kind, status.Supported, status.Entry, status.BuildUnverified = previewKindVite, false, previewKindVite, false
	status.Candidates = append([]string{previewKindVite}, status.Candidates...)
	if script != previewKindVite {
		return unavailableVitePreview(status, static, "preview_script_unsupported")
	}
	if _, err = previewViteModule(task.Worktree); err != nil {
		return unavailableVitePreview(status, static, "preview_dependencies_missing")
	}
	if runtime.GOOS != "windows" {
		return unavailableVitePreview(status, static, "preview_process_unsupported")
	}
	if _, err = exec.LookPath("node"); err != nil {
		return unavailableVitePreview(status, static, "preview_node_missing")
	}
	status.Supported, status.Reason = true, previewViteBasic
	return status
}

func unavailableVitePreview(status, fallback PreviewStatus, reason string) PreviewStatus {
	if fallback.BuildUnverified {
		return fallback
	}
	status.Reason = reason
	return status
}

func previewViteModule(candidate string) (string, error) {
	name := filepath.Join(candidate, "node_modules", previewKindVite, "dist", "node", "index.js")
	resolved, err := filepath.EvalSymlinks(name)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(candidate, resolved)
	if err != nil || !strings.HasPrefix(filepath.ToSlash(relative), "node_modules/") {
		return "", errors.New("preview_dependencies_missing: Vite must exist inside this candidate")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return "", errors.New("preview_dependencies_missing: installed Vite module is unavailable")
	}
	return resolved, nil
}

func (e *Engine) startVitePreview(ctx context.Context, task *Task, status PreviewStatus) (*PreviewStatus, error) {
	module, err := previewViteModule(task.Worktree)
	if err != nil {
		return nil, err
	}
	node, err := exec.LookPath("node")
	if err != nil {
		return nil, err
	}
	key := randomID()
	input, _ := json.Marshal(map[string]string{"module": module, "root": task.Worktree, "key": key})
	processCtx, cancel := context.WithCancel(storageContext(context.Background(), e.Root))
	log, err := newPreviewLog(e, task.ID, key, cancel)
	if err != nil {
		cancel()
		return nil, err
	}
	cmd := exec.CommandContext(processCtx, node, "--input-type=module", "-")
	cmd.Dir, cmd.Stdin, cmd.Stdout, cmd.Stderr = task.Worktree, strings.NewReader(vitePreviewScript+"\nawait start("+string(input)+");\n"), log, log
	cmd.Env = isolatedEnvironment(os.Environ(), map[string]string{"BROWSER": "none"}, []string{"NODE_OPTIONS", "NODE_PATH", "OPENAI_API_KEY", "DEEPSEEK_API_KEY", "DEEPSEEK_AUTH_TOKEN"})
	manager := e.previewManager()
	process := &previewProcess{done: make(chan struct{})}
	stop := previewProcessStop(cancel, process, log)
	entry, err := e.admitVitePreview(task, status, manager, stop, log)
	if err != nil {
		cancel()
		_ = log.Close()
		return nil, err
	}
	go func() {
		process.err = runOwnedCommand(processCtx, cmd)
		manager.mu.Lock()
		if manager.entries[task.ID] == entry && !entry.stopping {
			entry.status.Running, entry.status.Reason = false, "preview_process_exited"
		}
		manager.mu.Unlock()
		close(process.done)
	}()
	return finishVitePreview(ctx, manager, entry, process, log, key)
}

func (e *Engine) admitVitePreview(task *Task, status PreviewStatus, manager *PreviewManager, stop func() error, log *previewLog) (*managedPreview, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	fresh, err := e.Store.Get(task.ID)
	if err != nil || e.closing || e.active[task.ID] != nil || fresh.Updated != task.Updated {
		return nil, errors.New("preview_changed: candidate task changed before preview admission")
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.entries == nil {
		manager.entries = make(map[string]*managedPreview)
	}
	if manager.closed || len(manager.entries) >= 2 || manager.entries[task.ID] != nil {
		return nil, errors.New("preview_limit: stop an existing preview before starting this runtime")
	}
	entry, err := manager.newListener(e, task, status)
	if err == nil {
		entry.stop, entry.log = stop, log
		manager.entries[task.ID] = entry
	}
	return entry, err
}

func finishVitePreview(ctx context.Context, manager *PreviewManager, entry *managedPreview, process *previewProcess, log *previewLog, key string) (*PreviewStatus, error) {
	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()
	var ready previewReady
	select {
	case ready = <-log.ready:
	case <-process.done:
		_ = manager.StopTask(entry.status.TaskID)
		return nil, errors.New("preview_start_failed: installed Vite runtime exited; inspect its bounded preview log")
	case <-ctx.Done():
		_ = manager.StopTask(entry.status.TaskID)
		return nil, ctx.Err()
	case <-timeout.C:
		_ = manager.StopTask(entry.status.TaskID)
		return nil, errors.New("preview_start_timeout: the owned Vite runtime did not become ready")
	}
	target, _ := url.Parse("http://127.0.0.1:" + ready.Port)
	proxy := &httputil.ReverseProxy{Rewrite: func(request *httputil.ProxyRequest) {
		request.SetURL(target)
		request.Out.Host = target.Host
		request.Out.Header.Set("X-XBD-Preview-Key", key)
		request.Out.Header.Del("Origin")
	}}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.entries[entry.status.TaskID] != entry || entry.stopping {
		return nil, errors.New("preview_stopped: preview was stopped during runtime startup")
	}
	select {
	case <-process.done:
		return nil, errors.New("preview_start_failed: installed Vite runtime exited before admission completed")
	default:
	}
	entry.mu.Lock()
	entry.proxy = proxy
	entry.mu.Unlock()
	entry.status.Running, entry.status.Reason = true, previewViteBasic
	copy := entry.status
	return &copy, nil
}

func previewProcessStop(cancel context.CancelFunc, process *previewProcess, log *previewLog) func() error {
	return sync.OnceValue(func() error {
		cancel()
		select {
		case <-process.done:
			closeErr := log.Close()
			if process.err != nil && strings.Contains(process.err.Error(), "owned_process_cleanup:") {
				return errors.Join(process.err, closeErr)
			}
			return closeErr
		case <-time.After(5 * time.Second):
			return errors.New("preview_cleanup_unknown: owned runtime did not confirm exit")
		}
	})
}

func vitePreviewPathAllowed(candidate, name string) bool {
	if name == "" || name == "@vite/client" {
		return true
	}
	if strings.HasPrefix(name, "@fs/") {
		resolved, err := filepath.EvalSymlinks(filepath.FromSlash(strings.TrimPrefix(name, "@fs/")))
		if err != nil {
			return false
		}
		relative, err := filepath.Rel(candidate, resolved)
		if err != nil || !safeRelative(filepath.ToSlash(relative)) {
			return false
		}
		name = filepath.ToSlash(relative)
	}
	if strings.HasPrefix(name, "node_modules/") {
		return viteModulePathAllowed(name)
	}
	return resultPathAllowed(name)
}

func viteModulePathAllowed(name string) bool {
	if !safeRelative(name) || !viteRuntimeAsset(name) {
		return false
	}
	for _, part := range strings.Split(strings.ToLower(name), "/")[1:] {
		switch part {
		case "node_modules", ".pnpm", ".vite":
			continue
		default:
			if !resultComponentAllowed(part) {
				return false
			}
		}
	}
	return true
}

func viteRuntimeAsset(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".js", ".mjs", ".cjs", ".map", ".css", ".woff", ".woff2", ".png", ".svg":
		return true
	}
	return false
}

const vitePreviewScript = `import http from 'node:http';
import {pathToFileURL} from 'node:url';
async function start(input) {
  const {createServer} = await import(pathToFileURL(input.module).href);
  const vite = await createServer({configFile:false,root:input.root,logLevel:'warn',appType:'spa',esbuild:{jsx:'automatic'},server:{middlewareMode:true,hmr:false,cors:false,fs:{strict:true,allow:[input.root],deny:['**/.env','**/.env.*','**/.git/**','**/AGENTS.md','**/CLAUDE.md','**/*.{key,pem,p12,pfx}']}}});
  const server = http.createServer((req,res) => {
    if (req.headers['x-xbd-preview-key'] !== input.key || req.headers.host !== '127.0.0.1:'+server.address().port) {res.writeHead(403);res.end('Private preview upstream');return;}
    vite.middlewares(req,res);
  });
  server.headersTimeout=5000;server.requestTimeout=15000;
  server.listen(0,'127.0.0.1',()=>process.stdout.write(JSON.stringify({xbd_preview_ready:input.key,port:String(server.address().port)})+'\n'));
}
`
