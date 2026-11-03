package reliable

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

func (e *Engine) startLinkedBatch(id string, request messageRequest) (*TaskMessage, error) {
	request = normalizeAttachmentMessage(request)
	if err := validateMessage(request); err != nil {
		return nil, err
	}
	attachments, err := e.resolveAttachments(request.AttachmentIDs)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closing || e.active[id] != nil {
		return nil, errors.New("task_running: wait for this execution to finish before starting the next work batch")
	}
	parent, err := e.Store.Get(id)
	if err != nil {
		return nil, err
	}
	project, err := e.continuationProject(parent, request.Project)
	if err != nil {
		return nil, err
	}
	for _, old := range parent.Messages {
		if old.ID == request.ID {
			if old.Content != request.Content || old.Kind != request.Kind || !attachmentIDsMatch(old.Attachments, request.AttachmentIDs) || (old.TargetProject != "" && projectKey(old.TargetProject) != projectKey(project)) {
				return nil, errors.New("message_conflict: the message identifier was already used")
			}
			return &old, nil
		}
	}
	if parent.ArchivedAt != "" || (parent.State != stateInspected && parent.State != stateAccepted) {
		return nil, errors.New("new_batch_unavailable: finish investigation or accept the existing candidate first")
	}
	plan := modificationBatchPlan(parent)
	changedProject := projectKey(project) != projectKey(parent.Project)
	if changedProject {
		plan.FrontendPaths = ""
		plan.BackendPaths = ""
	}
	if plan == (ModelPlan{}) {
		plan = e.defaultPlan()
	}
	if err = e.validatePlan(plan); err != nil {
		return nil, err
	}
	child := &Task{ID: "next-" + digest([]byte(id + "\x00" + request.ID))[:26], ParentTaskID: id, PreviousContext: nextBatchContext(parent), Project: project, Goal: strings.TrimSpace(request.Content), ModelPlan: plan, PlanRevision: 1, State: stateQueued, ContinuePending: true, AutoSnapshot: true, AllowedPaths: parent.AllowedPaths, Commands: parent.Commands}
	if changedProject {
		child.AllowedPaths = []string{"."}
		child.Commands = nil
		child.PreviousContext = nextBatchProjectContext(parent, project)
	}
	child.Attachments = mergeAttachmentRefs(workAttachments(parent), questionReferenceAttachments(parent), attachments)
	if err = e.validateAttachments(child.Attachments); err != nil {
		return nil, err
	}
	if len(child.Commands) == 0 {
		child.Commands = discoverChecks(child.Project)
	}
	child.AutoChecks = len(child.Commands) == 0
	name := "model-plan-1-" + randomID() + ".json"
	if err = e.saveJSON(child.ID, name, plan); err != nil {
		return nil, err
	}
	child.PlanEvidence = []string{name}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	message := &TaskMessage{TargetProject: project, ID: request.ID, Kind: request.Kind, Content: request.Content, Status: "linked", LinkedTask: child.ID, Created: now, Updated: now, Attachments: attachments}
	err = e.Store.createLinked(parent.ID, child, message)
	return message, err
}

func (e *Engine) continuationProject(parent *Task, requested string) (string, error) {
	project := parent.Project
	if requested != "" {
		if !filepath.IsAbs(requested) {
			return "", errors.New("local_snapshot_directory: choose an absolute project folder")
		}
		resolved, err := filepath.EvalSymlinks(filepath.Clean(requested))
		if err != nil {
			return "", err
		}
		if !parent.ReadOnly && projectKey(resolved) != projectKey(parent.Project) {
			return "", errors.New("project_target_invalid: select a different project only after a read-only investigation")
		}
		project = resolved
	}
	if err := localProjectBoundary(project, e.Root); err != nil {
		return "", err
	}
	return project, nil
}
func nextBatchProjectContext(parent *Task, project string) json.RawMessage {
	var context map[string]any
	_ = json.Unmarshal(nextBatchContext(parent), &context)
	context["previous_project"] = parent.Project
	context["current_project"] = project
	context["context_scope"] = "Previous investigation concerns a different directory; independently verify all claims and source paths in the current project."
	raw, _ := json.Marshal(context)
	return raw
}

func nextBatchContext(parent *Task) json.RawMessage {
	var data map[string]any
	_ = json.Unmarshal(conversationContext(parent, ""), &data)
	delete(data, "previous_batch")
	data["previous_task_id"] = parent.ID
	raw, _ := json.Marshal(data)
	return raw
}

func (s *Store) createLinked(id string, child *Task, message *TaskMessage) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(tasksBucket)
		var parent Task
		if err := json.Unmarshal(bucket.Get([]byte(id)), &parent); err != nil {
			return err
		}
		if parent.ArchivedAt != "" || (parent.State != stateInspected && parent.State != stateAccepted) {
			return errors.New("new_batch_unavailable: the previous work batch changed")
		}
		for _, old := range parent.Messages {
			if old.ID == message.ID {
				return fmt.Errorf("message_conflict: %s", message.ID)
			}
		}
		if err := createTaskTx(tx, child); err != nil {
			return err
		}
		if err := bindProjectBatch(tx, parent.Project, message); err != nil {
			return err
		}
		parent.Messages = append(parent.Messages, *message)
		parent.Updated = message.Updated
		raw, err := json.Marshal(parent)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(id), raw)
	})
}
