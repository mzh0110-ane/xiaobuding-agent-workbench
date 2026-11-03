package reliable

import (
	"errors"
	"strings"
)

const (
	projectDeferred  = "deferred"
	projectCancelled = "cancelled"
)

type projectItemEdit struct {
	ID          string    `json:"id"`
	Title       *string   `json:"title,omitempty"`
	ShortTitle  *string   `json:"short_title,omitempty"`
	Acceptance  *string   `json:"acceptance,omitempty"`
	Milestone   *string   `json:"milestone,omitempty"`
	DependsOn   *[]string `json:"depends_on,omitempty"`
	Disposition *string   `json:"disposition,omitempty"`
	Reason      *string   `json:"reason,omitempty"`
}

func projectItemIndex(record *ProjectRecord, id string) int {
	for i := range record.Items {
		if record.Items[i].ID == id {
			return i
		}
	}
	return -1
}

func applyProjectItemEdit(record *ProjectRecord, edit *projectItemEdit) error {
	if edit == nil {
		return nil
	}
	index := projectItemIndex(record, edit.ID)
	if index < 0 {
		return errors.New("project_item_unavailable: only saved backlog items can be edited")
	}
	item := &record.Items[index]
	if item.TaskID != "" || !editableProjectStatus(item.Status) {
		return errors.New("project_item_conflict: assigned work and its acceptance history cannot be rewritten")
	}
	if err := patchProjectItemText(item, edit); err != nil {
		return err
	}
	if edit.DependsOn != nil {
		item.DependsOn = append([]string{}, (*edit.DependsOn)...)
	}
	return patchProjectDisposition(item, edit)
}

func editableProjectStatus(status string) bool {
	return status == projectPlanned || status == projectDeferred || status == projectCancelled
}

func patchProjectItemText(item *ProjectItem, edit *projectItemEdit) error {
	fields := []struct {
		input    *string
		target   *string
		limit    int
		required bool
	}{
		{edit.Title, &item.Title, 3000, true},
		{edit.ShortTitle, &item.ShortTitle, 240, false},
		{edit.Acceptance, &item.Acceptance, 4000, false},
		{edit.Milestone, &item.Milestone, 240, false},
	}
	for _, field := range fields {
		if field.input == nil {
			continue
		}
		value := strings.TrimSpace(*field.input)
		if !validOptionalProjectText(value, field.limit) || (field.required && value == "") {
			return errors.New("project_item_invalid: project item text is empty, invalid or oversized")
		}
		*field.target = value
	}
	return nil
}

func patchProjectDisposition(item *ProjectItem, edit *projectItemEdit) error {
	if edit.Disposition != nil {
		if !editableProjectStatus(*edit.Disposition) {
			return errors.New("project_item_invalid: completion is controlled by verification and human acceptance")
		}
		item.Status = *edit.Disposition
		if item.Status == projectPlanned {
			item.StatusReason = ""
		}
	}
	if edit.Reason != nil {
		item.StatusReason = strings.TrimSpace(*edit.Reason)
	}
	if item.Status == projectPlanned && item.StatusReason != "" {
		return errors.New("project_item_invalid: a disposition reason belongs to deferred or cancelled work")
	}
	if item.Status != projectPlanned && !validProjectText(item.StatusReason, 1000) {
		return errors.New("project_item_invalid: deferring or cancelling a requirement needs a reason")
	}
	return nil
}

func reorderProjectItems(record *ProjectRecord, order []string) error {
	if order == nil {
		return nil
	}
	if len(order) != len(record.Items) {
		return errors.New("project_item_order: supply every saved backlog item exactly once")
	}
	items := make([]ProjectItem, 0, len(order))
	seen := make(map[string]bool, len(order))
	for _, id := range order {
		index := projectItemIndex(record, id)
		if index < 0 || seen[id] {
			return errors.New("project_item_order: unknown or duplicated backlog item")
		}
		seen[id] = true
		items = append(items, record.Items[index])
	}
	record.Items = items
	return nil
}

func validateProjectDependencies(record *ProjectRecord) error {
	for _, item := range record.Items {
		if len(item.DependsOn) > 32 {
			return errors.New("project_dependency_invalid: at most 32 dependencies per item")
		}
		seen := make(map[string]bool, len(item.DependsOn))
		for _, id := range item.DependsOn {
			if id == item.ID || seen[id] || projectItemIndex(record, id) < 0 {
				return errors.New("project_dependency_invalid: dependencies must name distinct existing backlog items")
			}
			seen[id] = true
		}
	}
	visited := make(map[string]int, len(record.Items))
	for _, item := range record.Items {
		if projectDependencyCycle(record, item.ID, visited) {
			return errors.New("project_dependency_cycle: requirements cannot depend on each other cyclically")
		}
	}
	return nil
}

func projectDependencyCycle(record *ProjectRecord, id string, visited map[string]int) bool {
	if visited[id] == 1 {
		return true
	}
	if visited[id] == 2 {
		return false
	}
	visited[id] = 1
	for _, dependency := range record.Items[projectItemIndex(record, id)].DependsOn {
		if projectDependencyCycle(record, dependency, visited) {
			return true
		}
	}
	visited[id] = 2
	return false
}
