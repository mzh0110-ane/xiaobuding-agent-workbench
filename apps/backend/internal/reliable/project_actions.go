package reliable

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type projectEdit struct {
	ExpectedRevision int              `json:"expected_revision"`
	Items            []ProjectItem    `json:"items,omitempty"`
	Memory           *ProjectMemory   `json:"memory,omitempty"`
	AutoContinue     *bool            `json:"auto_continue,omitempty"`
	ItemEdit         *projectItemEdit `json:"item_edit,omitempty"`
	ItemOrder        []string         `json:"item_order,omitempty"`
}

func (e *Engine) EditProject(id string, edit projectEdit) (*ProjectRecord, error) {
	task, err := e.Store.Get(id)
	if err != nil {
		return nil, err
	}
	if task.ArchivedAt != "" {
		return nil, errors.New("task_archived: restore the task before editing its project")
	}
	_, err = e.Store.updateProject(task.Project, edit.ExpectedRevision, func(record *ProjectRecord) error {
		return editProjectRecord(record, task, edit)
	})
	if err != nil {
		return nil, err
	}
	if edit.AutoContinue != nil && *edit.AutoContinue {
		e.kickProjectNext(id)
	}
	return e.Project(id)
}

func editProjectRecord(record *ProjectRecord, task *Task, edit projectEdit) error {
	if len(record.Items)+len(edit.Items) > 200 || (edit.Memory != nil && len(record.Memories) >= 200) {
		return errors.New("project_limit: at most 200 project requirements and memories are supported")
	}
	if err := appendProjectItems(record, edit.Items, true, "user"); err != nil {
		return err
	}
	if err := applyProjectItemEdit(record, edit.ItemEdit); err != nil {
		return err
	}
	if err := reorderProjectItems(record, edit.ItemOrder); err != nil {
		return err
	}
	if err := validateProjectDependencies(record); err != nil {
		return err
	}
	if edit.Memory != nil {
		if err := appendProjectMemory(record, *edit.Memory, task); err != nil {
			return err
		}
	}
	if edit.AutoContinue != nil {
		record.AutoContinue = *edit.AutoContinue
	}
	return nil
}

func appendProjectItems(record *ProjectRecord, items []ProjectItem, confirmed bool, source string) error {
	for _, item := range items {
		if !validProjectText(item.Title, 3000) || !validOptionalProjectText(item.Acceptance, 4000) || item.TaskID != "" || item.Status != "" || len(item.Evidence) != 0 || item.StatusReason != "" {
			return errors.New("project_item_invalid: supply a title and optional acceptance condition, not a claimed completion")
		}
		if !validOptionalProjectText(item.ShortTitle, 240) || !validOptionalProjectText(item.Milestone, 240) {
			return errors.New("project_item_invalid: short titles and milestones must fit 240 bytes")
		}
		item.ID, item.Title, item.Acceptance = "item-"+randomID(), strings.TrimSpace(item.Title), strings.TrimSpace(item.Acceptance)
		item.ShortTitle, item.Milestone = strings.TrimSpace(item.ShortTitle), strings.TrimSpace(item.Milestone)
		item.Status, item.Source, item.Confirmed = projectPlanned, source, confirmed
		item.Evidence = []ProjectEvidence{}
		record.Items = append(record.Items, item)
	}
	return nil
}

func appendProjectMemory(record *ProjectRecord, memory ProjectMemory, task *Task) error {
	if !validProjectText(memory.Text, 4000) || (memory.Kind != "constraint" && memory.Kind != "decision" && memory.Kind != "open_issue") {
		return errors.New("project_memory_invalid: provide a decision, constraint or open issue of at most 4000 bytes")
	}
	if memory.Verified || memory.SourceTask != "" || memory.Source != "" || memory.Commit != "" {
		return errors.New("project_memory_invalid: user notes cannot claim independently verified provenance")
	}
	point := projectReferenceCommit(task)
	memory.ID, memory.Source, memory.SourceTask, memory.Commit = "memory-"+randomID(), "user", task.ID, point
	record.Memories = append(record.Memories, memory)
	return nil
}

func roundFile(round int, kind string) string {
	return fmt.Sprintf("round-%d-%s.json", round, kind)
}

func (e *Engine) StartProjectItem(id, itemID string) (*TaskMessage, error) {
	task, err := e.Store.Get(id)
	if err != nil {
		return nil, err
	}
	record, err := e.Project(id)
	if err != nil {
		return nil, err
	}
	for _, item := range record.Items {
		if item.ID == itemID && item.TaskID != "" {
			return &TaskMessage{LinkedTask: item.TaskID, Status: "linked"}, nil
		}
		if item.ID == itemID && item.TaskID == "" && item.Status == projectPlanned {
			return e.startProjectItem(task, item)
		}
	}
	return nil, errors.New("project_item_unavailable: the requirement is missing or already assigned to a batch")
}

func (e *Engine) startProjectItem(task *Task, item ProjectItem) (*TaskMessage, error) {
	if err := e.checkProjectDependencies(task.Project, item.ID); err != nil {
		return nil, err
	}
	goal := item.Title
	if item.Acceptance != "" {
		goal += "\n验收条件：\n" + item.Acceptance
	}
	request := messageRequest{ID: "project-" + digest([]byte(item.ID))[:26], Kind: messageRequirement, Content: goal}
	message, err := e.startLinkedBatch(task.ID, request)
	if err != nil {
		return nil, err
	}
	e.kickQuestions(message.LinkedTask)
	return message, nil
}

func bindProjectBatch(tx *bolt.Tx, project string, message *TaskMessage) error {
	if !strings.HasPrefix(message.ID, "project-") {
		return nil
	}
	record, err := readProject(tx, project)
	if err != nil {
		return err
	}
	for i := range record.Items {
		item := &record.Items[i]
		if message.ID != "project-"+digest([]byte(item.ID))[:26] {
			continue
		}
		goal := item.Title
		if item.Acceptance != "" {
			goal += "\n验收条件：\n" + item.Acceptance
		}
		if strings.TrimSpace(message.Content) != goal || item.TaskID != "" {
			return errors.New("project_item_conflict: the project requirement changed or was already assigned")
		}
		if item.Status != projectPlanned {
			return errors.New("project_item_conflict: the requirement is no longer planned")
		}
		if err := projectDependenciesAccepted(tx, record, *item); err != nil {
			return err
		}
		item.TaskID, item.Confirmed = message.LinkedTask, true
		record.Revision++
		raw, marshalErr := json.Marshal(record)
		if marshalErr != nil {
			return marshalErr
		}
		bucket, createErr := tx.CreateBucketIfNotExists(projectLedgerBucket)
		if createErr != nil {
			return createErr
		}
		return bucket.Put([]byte(record.ID), raw)
	}
	return nil
}

func (e *Engine) kickProjectNext(id string) {
	task, err := e.Store.Get(id)
	if err != nil || task.State != stateAccepted || task.ArchivedAt != "" {
		return
	}
	record, err := e.Project(id)
	if err != nil || !record.AutoContinue {
		return
	}
	for _, item := range record.Items {
		if item.TaskID == "" && item.Status == projectPlanned && item.Confirmed {
			if e.checkProjectDependencies(task.Project, item.ID) == nil {
				_, _ = e.startProjectItem(task, item)
				return
			}
		}
	}
}
