package reliable

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (h *HTTP) startTask(req createRequest) (*Task, error) {
	attachments, err := h.Engine.resolveAttachments(req.AttachmentIDs)
	if err != nil {
		return nil, err
	}
	plan := req.ModelPlan
	if plan == (ModelPlan{}) {
		plan = h.Engine.defaultPlan()
	}
	if err := h.Engine.validateTaskPlan(plan, req.ReadOnly); err != nil {
		return nil, err
	}
	task := &Task{ID: randomID(), Project: req.Project, Goal: strings.TrimSpace(req.Goal), Commands: req.Commands, AllowedPaths: req.AllowedPaths, State: stateQueued, AutoSnapshot: req.LocalSnapshot, Attachments: attachments}
	task.AutoChecks = req.LocalSnapshot && len(req.Commands) == 0
	task.ReadOnly = req.ReadOnly
	if req.ReadOnly {
		task.AutoChecks, task.AutoSnapshot = false, false
	}
	if err := h.Engine.Store.Create(task); err != nil {
		return nil, err
	}
	task, err = h.Engine.UpdateModelPlan(task.ID, 0, plan)
	if err != nil {
		return nil, err
	}
	go func() { _ = h.Engine.RunTask(context.Background(), task.ID) }()
	return task, nil
}

func (h *HTTP) setupAction(w http.ResponseWriter, r *http.Request, action string) {
	if action == "demo" {
		h.startDemo(w, r)
		return
	}
	var body struct{}
	if err := requestJSON(r, &body); err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if h.PickProject == nil {
		h.fail(w, errors.New("folder picker is unavailable; paste the project folder path"))
		return
	}
	const pickerTimeout = 3 * time.Minute
	// Human folder selection can outlast the server's ordinary API write timeout.
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(pickerTimeout + 5*time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		h.fail(w, fmt.Errorf("folder_picker_failed: %w", err))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), pickerTimeout)
	defer cancel()
	path, err := h.PickProject(ctx)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = fmt.Errorf("folder_picker_timeout: %w", err)
		} else {
			err = fmt.Errorf("folder_picker_failed: %w", err)
		}
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, map[string]string{"path": path})
}

func (h *HTTP) startDemo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ModelPlan ModelPlan `json:"model_plan"`
	}
	if err := requestJSON(r, &body); err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	template := h.DemoProject
	if body.ModelPlan.Mode != "" && h.TeamDemoProject != "" {
		template = h.TeamDemoProject
	}
	if template == "" {
		h.fail(w, errors.New("sample template is unavailable"))
		return
	}
	parent := filepath.Join(h.Engine.Root, "projects")
	if err := os.MkdirAll(parent, 0700); err != nil {
		h.fail(w, err)
		return
	}
	project := filepath.Join(parent, "sample-"+randomID())
	if _, err := git(r.Context(), parent, "clone", "--no-hardlinks", "--config", "core.autocrlf=false", "--", template, project); err != nil {
		h.fail(w, err)
		return
	}
	req := createRequest{Project: project, Goal: "修复 calculator.py 的 average：空列表返回 0，其他平均值保持正确，保留现有测试，只修改 calculator.py。", Commands: []string{"python -m unittest discover -s tests -v"}, AllowedPaths: []string{"calculator.py"}, ModelPlan: body.ModelPlan}
	if body.ModelPlan.Mode != "" {
		req = teamDemoRequest(project, body.ModelPlan)
	}
	task, err := h.startTask(req)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusAccepted, task)
}

func teamDemoRequest(project string, plan ModelPlan) createRequest {
	req := createRequest{Project: project, ModelPlan: plan}
	req.ModelPlan.FrontendPaths, req.ModelPlan.BackendPaths = "", ""
	if plan.Mode != domainBackend {
		req.Goal = "前端：修复 frontend/app.js 的 parseValues，保留负数，空输入返回空数组，保留页面与交互。"
		req.Commands = append(req.Commands, "node --test tests/frontend.test.cjs")
		req.AllowedPaths = append(req.AllowedPaths, "frontend/app.js")
		req.ModelPlan.FrontendPaths = "frontend/app.js"
	}
	if plan.Mode != domainFrontend {
		req.Goal += " 后端：修复 backend/calculator.py 的 average，空列表返回0，其他平均值保持正确。"
		req.Commands = append(req.Commands, "python -m unittest discover -s tests -v")
		req.AllowedPaths = append(req.AllowedPaths, "backend/calculator.py")
		req.ModelPlan.BackendPaths = "backend/calculator.py"
	}
	req.Goal += " 只修改本次分工对应文件，保留所有现有测试。未启用的工种及其已知问题不属于本次目标。"
	return req
}
