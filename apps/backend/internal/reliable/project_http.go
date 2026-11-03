package reliable

import (
	"errors"
	"net/http"
	"strings"
)

func (h *HTTP) projectRoute(w http.ResponseWriter, r *http.Request, id string, parts []string) {
	if len(parts) == 0 {
		h.projectRecordRoute(w, r, id)
		return
	}
	if len(parts) != 1 || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var result any
	var err error
	switch parts[0] {
	case "start":
		var body struct {
			ItemID string `json:"item_id"`
		}
		if err = requestJSON(r, &body); err == nil {
			result, err = h.Engine.StartProjectItem(id, body.ItemID)
		}
	case "plan":
		var body struct{}
		if err = requestJSON(r, &body); err == nil {
			result, err = h.Engine.PlanProject(id)
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, result)
}

func (h *HTTP) projectRecordRoute(w http.ResponseWriter, r *http.Request, id string) {
	var result any
	var err error
	switch r.Method {
	case http.MethodGet:
		result, err = h.Engine.Project(id)
	case http.MethodPost:
		var edit projectEdit
		if err = requestJSON(r, &edit); err == nil {
			result, err = h.Engine.EditProject(id, edit)
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, result)
}

func (h *HTTP) storageRoute(w http.ResponseWriter, r *http.Request, path string) {
	var result any
	var err error
	switch {
	case path == "storage" && r.Method == http.MethodGet:
		result, err = h.Engine.StorageUsage(r.Context())
	case path == "storage/cleanup" && r.Method == http.MethodPost:
		var body struct {
			DryRun *bool `json:"dry_run"`
		}
		if err = requestJSON(r, &body); err == nil {
			if body.DryRun == nil {
				err = errors.New("storage_cleanup_confirmation: explicitly select preview or cleanup")
			} else {
				result, err = h.Engine.CleanupTemporary(r.Context(), *body.DryRun)
			}
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, result)
}

func (e *Engine) PlanProject(id string) (*TaskMessage, error) {
	task, err := e.Store.Get(id)
	if err != nil {
		return nil, err
	}
	context, err := e.projectPrompt(task)
	if err != nil {
		return nil, err
	}
	content := "请结合项目记录和当前源代码，将项目长期目标整理成最多 12 个可独立验收的后续任务，在 plan 中逐条写出具体目标和验收条件。已完成内容不要重复列为待办。明确标出阻塞问题。这次只读调查，不实施修改；建议需用户确认后执行。\n" + context
	message, err := e.submitMessage(id, messageRequest{ID: "project-plan-" + randomID(), Kind: messageQuestion, Content: content})
	if err == nil {
		e.kickQuestions(id)
	}
	return message, err
}

func projectPlanMessage(id string) bool { return strings.HasPrefix(id, "project-plan-") }
