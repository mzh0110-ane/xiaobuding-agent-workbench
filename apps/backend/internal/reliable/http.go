package reliable

import (
	"context"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed page.html
var page string

//go:embed page.css
var pageCSS string

//go:embed page-studio.css
var pageStudioCSS string

//go:embed page-state.js
var pageStateJS string

//go:embed page-plan.js
var pagePlanJS string

//go:embed page.js
var pageJS string

//go:embed page-desktop.js
var pageDesktopJS string

//go:embed page-appearance.js
var pageAppearanceJS string

//go:embed page-proof.js
var pageProofJS string

//go:embed page-activity.js
var pageActivityJS string

//go:embed page-conversation.js
var pageConversationJS string

//go:embed page-attachment-state.js
var pageAttachmentStateJS string

//go:embed page-attachments.js
var pageAttachmentsJS string

//go:embed page-agent-settings.js
var pageAgentSettingsJS string

//go:embed page-workspace.js
var pageWorkspaceJS string

//go:embed page-project.js
var pageProjectJS string

//go:embed page-workbench-state.js
var pageWorkbenchStateJS string

//go:embed page-assistant-state.js
var pageAssistantStateJS string

//go:embed page-assistants.js
var pageAssistantsJS string

//go:embed page-context.js
var pageContextJS string

//go:embed page-workbench.js
var pageWorkbenchJS string

//go:embed page-artifacts.js
var pageArtifactsJS string

//go:embed page-budget.js
var pageBudgetJS string

//go:embed assets/codex-app-ga-logo-3e5209898ca3.png
var codexBrand []byte

//go:embed assets/deepseek-official.png
var deepseekBrand []byte

//go:embed assets/zcode-official.png
var zcodeBrand []byte

//go:embed assets/pudding-avatar.png
var puddingAvatar []byte

//go:embed assets/teahouse-background.png
var teahouseBackground []byte

type HTTP struct {
	catalogMu                 sync.Mutex
	ReloadCatalog             func() ([]Provider, error)
	Engine                    *Engine
	TeamDemoProject           string
	Token, Model, DemoProject string
	PickProject               func(context.Context) (string, error)
	OpenFolder                func(context.Context, string) error
}

func (h *HTTP) localRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return false
	}
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !net.ParseIP(peer).IsLoopback() {
		return false
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host != r.Host || parsed.Scheme != "http" {
			return false
		}
	}
	return true
}

func (h *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "same-origin")
	if !h.localRequest(r) {
		http.Error(w, "Local same-origin access required", http.StatusForbidden)
		return
	}
	if !h.authorizedMutation(w, r) {
		return
	}
	if picture := desktopImage(r.URL.Path); picture != nil && r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(picture)
		return
	}
	if r.URL.Path == "/reliable" || r.URL.Path == "/reliable/" {
		h.servePage(w, r)
		return
	}
	h.dispatch(w, r)
}

func (h *HTTP) authorizedMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-XBD-Token")), []byte(h.Token)) != 1 || h.Token == "" {
			http.Error(w, "Explicit browser action required", http.StatusForbidden)
			return false
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			if r.Method == http.MethodPost && r.URL.Path == "/api/v1/reliable/attachments" && strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
				return true
			}
			http.Error(w, "JSON required", http.StatusBadRequest)
			return false
		}
	}
	return true
}

func (h *HTTP) servePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' blob:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'")
	body := strings.NewReplacer("__TOKEN__", h.Token, "__STYLE__", pageCSS+"\n"+pageStudioCSS, "__SCRIPT__", pageStateJS+"\n"+pageWorkbenchStateJS+"\n"+pageAssistantStateJS+"\n"+pagePlanJS+"\n"+pageAppearanceJS+"\n"+pageDesktopJS+"\n"+pageArtifactsJS+"\n"+pageProofJS+"\n"+pageActivityJS+"\n"+pageAttachmentStateJS+"\n"+pageAttachmentsJS+"\n"+pageConversationJS+"\n"+pageAgentSettingsJS+"\n"+pageWorkspaceJS+"\n"+pageBudgetJS+"\n"+pageProjectJS+"\n"+pageWorkbenchJS+"\n"+pageAssistantsJS+"\n"+pageContextJS+"\n"+pageJS).Replace(page)
	_, _ = io.WriteString(w, body)
}

func (h *HTTP) dispatch(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/reliable/")
	if path == "storage/budget" {
		h.storageBudgetRoute(w, r)
		return
	}
	if path == "storage" || strings.HasPrefix(path, "storage/") {
		h.storageRoute(w, r, path)
		return
	}
	if path == "attachments" || strings.HasPrefix(path, "attachments/") {
		h.attachmentRoute(w, r, strings.Split(path, "/")[1:])
		return
	}
	if path == "models/refresh" && r.Method == http.MethodPost {
		h.refreshCatalog(w, r)
		return
	}
	if r.Method == http.MethodPost && (path == "demo" || path == "pick-project") {
		h.setupAction(w, r, path)
		return
	}
	if path == "config" && r.Method == http.MethodGet {
		h.reply(w, 200, map[string]any{"app": "xiaobuding", "version": "1.23.3", "model": h.Model, "max_rounds": MaxRounds, "demo_project": h.DemoProject, "providers": h.Engine.providerList(), "default_plan": h.Engine.defaultTeamPlan()})
		return
	}
	if path == "tasks" {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("view") == "summary" {
				h.summaryList(w, r)
				return
			}
			tasks, err := h.Engine.Store.List()
			if err != nil {
				h.fail(w, err)
				return
			}
			h.reply(w, 200, tasks)
			return
		}
		if r.Method == http.MethodPost {
			h.create(w, r)
			return
		}
	}
	parts := strings.Split(path, "/")
	h.dispatchTask(w, r, parts)
}

func (h *HTTP) dispatchTask(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 2 || parts[0] != "tasks" || !safeRelative(parts[1]) {
		http.NotFound(w, r)
		return
	}
	id := parts[1]
	if _, err := h.Engine.Store.Get(id); err != nil {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 4 && parts[2] == "context" && parts[3] == "status" {
		h.sharedContextStatusRoute(w, r, id)
		return
	}
	if len(parts) >= 3 && parts[2] == "project" {
		h.projectRoute(w, r, id, parts[3:])
		return
	}
	if len(parts) == 3 && h.workbenchTaskRoute(w, r, id, parts[2]) {
		return
	}
	if len(parts) == 2 && r.Method == http.MethodGet {
		task, _ := h.Engine.Store.Get(id)
		if r.URL.Query().Get("view") == "workspace" {
			h.workspaceDetail(w, r, task)
			return
		}
		h.reply(w, 200, task)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet && parts[2] == "messages" {
		task, _ := h.Engine.Store.Get(id)
		h.messagePage(w, r, task)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet && parts[2] == "activity" {
		if h.activityHistoryRoute(w, r, id) {
			return
		}
		after, err := activityCursor(r.URL.Query().Get("after"))
		if err != nil {
			h.reply(w, 400, map[string]string{"error": err.Error()})
			return
		}
		result, err := h.Engine.readActivity(id, after)
		if err != nil {
			h.fail(w, err)
			return
		}
		h.reply(w, 200, result)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet && parts[2] == "evidence" {
		h.evidence(w, r, id)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodPost {
		if parts[2] == "exhausted-next-batch" {
			h.startExhaustedBatch(w, r, id)
			return
		}
		if parts[2] == "future-models" {
			h.updateFutureModels(w, r, id)
			return
		}
		if parts[2] == "messages" {
			h.submitMessage(w, r, id)
			return
		}
		if parts[2] == "models" {
			h.updateModels(w, r, id)
			return
		}
		h.action(w, r, id, parts[2])
		return
	}
	http.NotFound(w, r)
}

func (h *HTTP) reply(w http.ResponseWriter, status int, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "Response encoding failed", 500)
		return
	}
	if h.Engine.Redact != nil {
		raw = h.Engine.Redact(raw)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func (h *HTTP) fail(w http.ResponseWriter, err error) {
	h.reply(w, http.StatusConflict, map[string]string{"error": err.Error()})
}

func requestJSON(r *http.Request, out any) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, 65537))
	if err != nil {
		return err
	}
	if len(data) > 65536 {
		return errors.New("request exceeds 64 KiB")
	}
	return decodeStrict(data, out)
}

type createRequest struct {
	AttachmentIDs []string  `json:"attachment_ids,omitempty"`
	ReadOnly      bool      `json:"read_only"`
	LocalSnapshot bool      `json:"-"`
	ModelPlan     ModelPlan `json:"model_plan"`
	Project       string    `json:"project"`
	Goal          string    `json:"goal"`
	Commands      []string  `json:"commands"`
	AllowedPaths  []string  `json:"allowed_paths"`
}

func (h *HTTP) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := requestJSON(r, &req); err != nil {
		h.reply(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Goal) == "" && len(req.AttachmentIDs) > 0 {
		req.Goal = "请根据所附文件完成分析，并先确认需要修改的内容。"
	}
	if err := validateCreateInput(req); err != nil {
		h.reply(w, 400, map[string]string{"error": err.Error()})
		return
	}
	project, err := filepath.EvalSymlinks(filepath.Clean(req.Project))
	if err != nil {
		h.fail(w, err)
		return
	}
	if err = selectedProjectBoundary(project, h.Engine.Root, req.ReadOnly); err != nil {
		h.fail(w, err)
		return
	}
	if len(req.Commands) == 0 && !req.ReadOnly {
		req.Commands = discoverChecks(project)
	}
	if len(req.AllowedPaths) == 0 {
		req.AllowedPaths = []string{"."}
	}
	req.Project = project
	req.LocalSnapshot = !req.ReadOnly
	task, err := h.startTask(req)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusAccepted, task)
}

func validateCreateInput(req createRequest) error {
	if strings.TrimSpace(req.Goal) == "" || len(req.Goal) > 12000 || !filepath.IsAbs(req.Project) {
		return errors.New("provide an absolute project folder path and a concrete goal")
	}
	if len(req.Commands) > 12 {
		return errors.New("at most 12 verification commands are allowed")
	}
	for _, command := range req.Commands {
		if strings.TrimSpace(command) == "" || strings.ContainsAny(command, "\r\n\x00") || len(command) > 2000 {
			return errors.New("each verification command must be a nonempty single line")
		}
	}
	for _, path := range req.AllowedPaths {
		if path != "." && !safeRelative(path) {
			return errors.New("modification scope must use relative files or directories")
		}
	}
	return nil
}

func (h *HTTP) action(w http.ResponseWriter, r *http.Request, id, action string) {
	var body struct{}
	if err := requestJSON(r, &body); err != nil {
		h.reply(w, 400, map[string]string{"error": err.Error()})
		return
	}
	var err error
	switch action {
	case "accept":
		err = h.Engine.Accept(r.Context(), id)
	case "cancel":
		err = h.Engine.Cancel(id)
	case "recover-cancelled":
		var task *Task
		task, err = h.Engine.RecoverCancelled(id)
		if err == nil {
			h.reply(w, http.StatusAccepted, task)
			return
		}
	case "archive", "unarchive":
		err = h.Engine.ArchiveTask(id, action == "archive")
	case "resume":
		t, loadErr := h.Engine.Store.Get(id)
		if loadErr != nil {
			err = loadErr
			break
		}
		if t.ArchivedAt != "" {
			err = errors.New("task_archived: restore the task before resuming")
			break
		}
		if (t.State != stateBlocked && t.State != stateInterrupted) || (t.Round >= MaxRounds && !acceptancePending(t) && !domainStageResumeEligible(t)) {
			err = errors.New("this task cannot resume within its three-round limit")
			break
		}
		go func() { _ = h.Engine.RunTask(context.Background(), id) }()
	case desktopOpen, "open-project", "open-evidence":
		err = h.openTaskFolder(r.Context(), id, action)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	task, err := h.Engine.Store.Get(id)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, 200, task)
}

func (h *HTTP) evidence(w http.ResponseWriter, r *http.Request, id string) {
	root := filepath.Join(h.Engine.Root, "evidence", id)
	if path := r.URL.Query().Get("path"); path != "" {
		h.evidenceFile(w, r, root, path)
		return
	}
	h.evidenceList(w, root)
}

func (h *HTTP) evidenceFile(w http.ResponseWriter, r *http.Request, root, path string) {
	if !safeRelative(path) {
		h.reply(w, 400, map[string]string{"error": "Invalid evidence path"})
		return
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || !filepath.IsLocal(rel) {
		h.reply(w, 400, map[string]string{"error": "Evidence path escaped its task"})
		return
	}
	file, err := os.Open(resolved)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, 20<<20))
	if err != nil {
		h.fail(w, err)
		return
	}
	if h.Engine.Redact != nil {
		data = h.Engine.Redact(data)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

func (h *HTTP) evidenceList(w http.ResponseWriter, root string) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if os.IsNotExist(walkErr) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !d.IsDir() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		if len(files) > 1000 {
			return fmt.Errorf("evidence listing exceeds 1000 files")
		}
		return nil
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, 200, files)
}
