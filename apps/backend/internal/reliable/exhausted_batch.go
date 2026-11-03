package reliable

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

type ExhaustedSeed struct {
	ParentID           string    `json:"parent_id"`
	ParentIdentity     string    `json:"parent_identity"`
	ExpectedSourceHash string    `json:"expected_source_hash,omitempty"`
	Intent             string    `json:"intent"`
	InitialPlan        ModelPlan `json:"initial_plan"`
	ParentReason       string    `json:"parent_reason,omitempty"`
}

func exhaustedBatchEligible(t *Task) error {
	if t.ArchivedAt != "" {
		return errors.New("batch_next_unavailable: restore the previous batch before starting a continuation")
	}
	return exhaustedRetainedParent(t)
}

func exhaustedRetainedParent(t *Task) error {
	if t.ReadOnly || t.Round != MaxRounds || t.Baseline == nil || t.Worktree == "" {
		return errors.New("batch_next_unavailable: choose an unarchived modification batch that has used three rounds")
	}
	switch t.State {
	case stateExhausted, stateCancelled, stateBlocked, stateInterrupted:
		return nil
	default:
		return errors.New("batch_next_unavailable: stop the current execution before starting another batch")
	}
}

func (e *Engine) StartExhaustedBatch(id string, request messageRequest) (*TaskMessage, error) {
	attachments, err := e.resolveAttachments(request.AttachmentIDs)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closing || e.active[id] != nil {
		return nil, errors.New("task_running: wait for the current execution to finish")
	}
	parent, err := e.Store.Get(id)
	if err != nil {
		return nil, err
	}
	if request, err = normalizeExhaustedRequest(request); err != nil {
		return nil, err
	}
	prior, err := exhaustedAttachmentReplay(parent, request)
	if err != nil || prior != nil {
		return prior, err
	}
	if err = exhaustedBatchEligible(parent); err != nil {
		return nil, err
	}
	plan := continuationPlan(parent)
	if plan == (ModelPlan{}) {
		plan = e.defaultPlan()
	}
	if err = e.validatePlan(plan); err != nil {
		return nil, err
	}
	child := newExhaustedChild(parent, request, plan)
	child.Attachments = mergeAttachmentRefs(workAttachments(parent), questionReferenceAttachments(parent), attachments)
	if err = e.validateAttachments(child.Attachments); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	message := &TaskMessage{ID: request.ID, Kind: request.Kind, Content: request.Content, Status: "linked", LinkedTask: child.ID, Created: now, Updated: now, Attachments: attachments}
	if err = e.Store.createExhaustedLinked(parent, child, message); err != nil {
		return nil, err
	}
	return message, nil
}

func exhaustedAttachmentReplay(parent *Task, request messageRequest) (*TaskMessage, error) {
	for _, old := range parent.Messages {
		if old.ID != request.ID {
			continue
		}
		if old.Kind != request.Kind || old.Content != request.Content || old.LinkedTask == "" || !attachmentIDsMatch(old.Attachments, request.AttachmentIDs) {
			return nil, errors.New("message_conflict: the message identifier was already used")
		}
		return &old, nil
	}
	return nil, nil
}

func normalizeExhaustedRequest(request messageRequest) (messageRequest, error) {
	if request.Kind == "" {
		request.Kind = messageRequirement
	}
	if request.Kind != messageRequirement {
		return request, errors.New("message_invalid: a continuation requires an explicit modification request")
	}
	if strings.TrimSpace(request.Content) == "" {
		request.Content = "继续原有需求，并修复上一批独立审查中尚未解决的问题。"
	}
	return request, validateMessage(request)
}

func newExhaustedChild(parent *Task, request messageRequest, plan ModelPlan) *Task {
	id := "next-" + digest([]byte(parent.ID + "\x00" + request.ID))[:26]
	seed := &ExhaustedSeed{ParentID: parent.ID, ParentIdentity: exhaustedParentIdentity(parent), ExpectedSourceHash: parent.Snapshot, Intent: "seed-" + randomID() + "-intent.json", InitialPlan: plan, ParentReason: parent.Reason}
	commands := append([]string(nil), verificationCommands(parent)...)
	if parent.AutoChecks {
		commands = nil
	}
	goal := taskGoal(parent) + "\n\nNEXT BATCH REQUIREMENT:\n" + strings.TrimSpace(request.Content)
	return &Task{ID: id, ParentTaskID: parent.ID, PreviousContext: nextBatchContext(parent), Project: parent.Project, Goal: goal,
		ModelPlan: plan, PlanRevision: 1, PlanEvidence: []string{"model-plan-1-" + randomID() + ".json"}, State: stateQueued,
		ContinuePending: true, Commands: commands, AutoChecks: parent.AutoChecks || len(commands) == 0, AllowedPaths: append([]string(nil), parent.AllowedPaths...),
		ApprovedPaths: append([]string(nil), parent.ApprovedPaths...), SeedPending: seed}
}

func (s *Store) createExhaustedLinked(expected, child *Task, message *TaskMessage) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(tasksBucket)
		var parent Task
		if err := json.Unmarshal(bucket.Get([]byte(expected.ID)), &parent); err != nil {
			return err
		}
		if err := exhaustedBatchEligible(&parent); err != nil {
			return err
		}
		if parent.SuccessorTaskID != "" {
			return fmt.Errorf("batch_successor_exists: continue the linked batch %s", parent.SuccessorTaskID)
		}
		if exhaustedParentIdentity(&parent) != child.SeedPending.ParentIdentity {
			return errors.New("batch_parent_changed: the previous batch changed during admission")
		}
		parent.SuccessorTaskID = child.ID
		parent.Messages = append(parent.Messages, *message)
		parent.Updated = message.Updated
		raw, err := json.Marshal(&parent)
		if err != nil {
			return err
		}
		if err = bucket.Put([]byte(parent.ID), raw); err != nil {
			return err
		}
		return createExhaustedChildTx(tx, &parent, child)
	})
}

func createExhaustedChildTx(tx *bolt.Tx, parent, child *Task) error {
	bucket := tx.Bucket(tasksBucket)
	if bucket.Get([]byte(child.ID)) != nil {
		return errors.New("task already exists")
	}
	if err := bucket.ForEach(func(_, raw []byte) error {
		var old Task
		if err := json.Unmarshal(raw, &old); err != nil {
			return err
		}
		if old.ID != parent.ID && projectsOverlap(old.Project, child.Project) && old.State != stateAccepted && old.State != stateCancelled && old.State != stateInspected && !supersededProjectTask(tx, &old) {
			return errors.New("batch_project_conflict: another unfinished batch owns this project")
		}
		return nil
	}); err != nil {
		return err
	}
	child.Created = time.Now().UTC().Format(time.RFC3339Nano)
	child.Updated = child.Created
	raw, err := json.Marshal(child)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(child.ID), raw)
}

func supersededProjectTask(tx *bolt.Tx, t *Task) bool {
	if t.SuccessorTaskID == "" || exhaustedRetainedParent(t) != nil {
		return false
	}
	var next Task
	if json.Unmarshal(tx.Bucket(tasksBucket).Get([]byte(t.SuccessorTaskID)), &next) != nil {
		return false
	}
	return next.ParentTaskID == t.ID && projectKey(next.Project) == projectKey(t.Project) && !next.ReadOnly &&
		((next.SeedPending != nil && next.SeedPending.ParentID == t.ID) || len(next.SeedEvidence) > 0)
}

func exhaustedParentIdentity(t *Task) string {
	copy := *t
	copy.Messages, copy.Updated, copy.SuccessorTaskID = nil, "", ""
	copy.FuturePlan = nil
	copy.ArchivedAt, copy.State, copy.Reason = "", "", ""
	raw, _ := json.Marshal(copy)
	return digest(raw)
}
