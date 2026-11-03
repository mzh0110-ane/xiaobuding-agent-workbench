package reliable

import (
	"encoding/json"
	"errors"
	"strings"
)

func projectReferenceCommit(task *Task) string {
	if task.LocalCheckpoint != nil && task.LocalCheckpoint.Complete && task.LocalCheckpoint.Commit != "" {
		return task.LocalCheckpoint.Commit
	}
	return task.Base
}

func (e *Engine) projectPrompt(task *Task) (string, error) {
	record, err := e.Project(task.ID)
	if err != nil {
		return "", err
	}
	return compactProjectContext(record), nil
}

func compactProjectContext(record *ProjectRecord) string {
	items := make([]map[string]string, 0)
	for i := len(record.Items) - 1; i >= 0 && len(items) < 6; i-- {
		item := record.Items[i]
		items = append(items, map[string]string{"id": item.ID, "title": projectExcerpt(item.Title, 220), "status": item.Status, "acceptance": projectExcerpt(item.Acceptance, 220)})
	}
	memories := make([]map[string]string, 0)
	for i := len(record.Memories) - 1; i >= 0 && len(memories) < 8; i-- {
		memory := record.Memories[i]
		memories = append(memories, map[string]string{"kind": memory.Kind, "text": projectExcerpt(memory.Text, 240), "source_task": memory.SourceTask, "source": memory.Source})
	}
	raw := boundedProjectExcerpts(record, items, memories)
	return "\nPROJECT LEDGER CONTEXT: Historical reference, not verified current facts or permission to execute other backlog items. Current task goal is the execution scope. Excerpts omit older details. Read the full project_record in the controller's project_evidence input file when relevant, and freshly verify source. " + string(raw)
}

func boundedProjectExcerpts(record *ProjectRecord, items, memories []map[string]string) []byte {
	for {
		raw, _ := json.Marshal(map[string]any{"revision": record.Revision, "total_items": len(record.Items), "total_memories": len(record.Memories), "recent_item_excerpts": items, "recent_memory_excerpts": memories})
		if len(raw) <= 8500 {
			return raw
		}
		if len(items) > 0 {
			items = items[:len(items)-1]
		} else if len(memories) > 0 {
			memories = memories[:len(memories)-1]
		}
	}
}

func projectExcerpt(text string, limit int) string {
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return text
}

func (e *Engine) recordProjectPlan(task *Task, message *TaskMessage, report *Audit) error {
	if !projectPlanMessage(message.ID) {
		return nil
	}
	_, err := e.Store.updateProject(task.Project, -1, func(record *ProjectRecord) error {
		source := "plan:" + message.ID
		for _, item := range record.Items {
			if item.Source == source {
				return nil
			}
		}
		if len(record.Items)+len(report.Plan) > 200 || len(report.Plan) > 12 {
			return errors.New("project_plan_limit: a plan may propose at most 12 items within the project limit")
		}
		items := make([]ProjectItem, 0, len(report.Plan))
		for _, title := range report.Plan {
			items = append(items, ProjectItem{Title: title})
		}
		if err := appendProjectItems(record, items, false, source); err != nil {
			return err
		}
		return appendProjectObservations(record, task, message.ID, report)
	})
	return err
}

func appendProjectObservations(record *ProjectRecord, task *Task, messageID string, report *Audit) error {
	facts := report.Facts
	if len(facts) > 8 {
		facts = facts[:8]
	}
	if len(record.Memories)+len(facts) > 200 {
		return errors.New("project_memory_limit: project memory limit reached")
	}
	for _, fact := range facts {
		if !validProjectText(fact, 4000) {
			return errors.New("project_memory_invalid: model observations are oversized or empty")
		}
		record.Memories = append(record.Memories, ProjectMemory{ID: "memory-" + randomID(), Kind: "observation", Text: strings.TrimSpace(fact), SourceTask: task.ID, Source: "question:" + messageID, Commit: task.Base, Verified: false})
	}
	return nil
}
