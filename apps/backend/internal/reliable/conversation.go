package reliable

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	messageQuestion    = "question"
	messageRequirement = "requirement"
	messageReading     = "reading"
)

type TaskMessage struct {
	TargetProject      string       `json:"target_project,omitempty"`
	ContextHash        string       `json:"context_hash,omitempty"`
	ContextAttachments []Attachment `json:"context_attachments,omitempty"`
	Attachments        []Attachment `json:"attachments,omitempty"`
	ID                 string       `json:"id"`
	Kind               string       `json:"kind"`
	Content            string       `json:"content"`
	Status             string       `json:"status"`
	Created            string       `json:"created"`
	Updated            string       `json:"updated"`
	Provider           string       `json:"provider,omitempty"`
	Model              string       `json:"model,omitempty"`
	Effort             string       `json:"effort,omitempty"`
	Session            string       `json:"session,omitempty"`
	Directory          string       `json:"directory,omitempty"`
	Evidence           string       `json:"evidence,omitempty"`
	Error              string       `json:"error,omitempty"`
	Reply              *Audit       `json:"reply,omitempty"`
	Revision           int          `json:"revision,omitempty"`
	AppliedRound       int          `json:"applied_round,omitempty"`
	LinkedTask         string       `json:"linked_task,omitempty"`
}

type messageRequest struct {
	Project       string   `json:"project,omitempty"`
	AttachmentIDs []string `json:"attachment_ids,omitempty"`
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Content       string   `json:"content"`
}

func (h *HTTP) submitMessage(w http.ResponseWriter, r *http.Request, id string) {
	var request messageRequest
	if err := requestJSON(r, &request); err != nil {
		h.reply(w, 400, map[string]string{"error": err.Error()})
		return
	}
	message, err := h.Engine.submitMessage(id, request)
	if errors.Is(err, errNewBatch) {
		message, err = h.Engine.startLinkedBatch(id, request)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.Engine.kickQuestions(id)
	if message.LinkedTask != "" {
		h.Engine.kickQuestions(message.LinkedTask)
	}
	h.reply(w, http.StatusAccepted, message)
}

func validateMessage(request messageRequest) error {
	if !safeRelative(request.ID) || strings.ContainsAny(request.ID, "/\\") || len(request.ID) > 80 {
		return errors.New("message_invalid: invalid message identifier")
	}
	if strings.TrimSpace(request.Content) == "" || len(request.Content) > 12000 || !utf8.ValidString(request.Content) || strings.ContainsRune(request.Content, 0) {
		return errors.New("message_invalid: provide a message of at most 12000 UTF-8 bytes")
	}
	if request.Project != "" && request.Kind != messageRequirement {
		return errors.New("project_target_invalid: target folder is only available for modification requests")
	}
	if request.Kind != messageQuestion && request.Kind != messageRequirement {
		return errors.New("message_invalid: unsupported message kind")
	}
	return nil
}

func (e *Engine) submitMessage(id string, request messageRequest) (*TaskMessage, error) {
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
	if e.closing {
		return nil, errors.New("application is shutting down")
	}
	var result TaskMessage
	_, err = e.Store.Update(id, func(saved *Task) error {
		if request.Project != "" && !saved.ReadOnly {
			return errors.New("project_target_invalid: changing the target folder requires a read-only investigation")
		}
		for _, message := range saved.Messages {
			if message.ID == request.ID {
				if message.Content != request.Content || message.Kind != request.Kind || !attachmentIDsMatch(message.Attachments, request.AttachmentIDs) || (request.Project != "" && message.TargetProject != "" && projectKey(request.Project) != projectKey(message.TargetProject)) {
					return errors.New("message_conflict: the message identifier was already used")
				}
				result = message
				return nil
			}
		}
		if saved.ArchivedAt != "" {
			return errors.New("task_archived: restore the task before sending messages")
		}
		if request.Kind == messageRequirement && (saved.ReadOnly || saved.State == stateAccepted) {
			return errNewBatch
		}
		if len(saved.Messages) >= 200 {
			return errors.New("message_limit: start a linked work batch to continue")
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		result = TaskMessage{ID: request.ID, Kind: request.Kind, Content: request.Content, Status: stateQueued, Created: now, Updated: now, Attachments: attachments}
		if request.Kind == messageRequirement {
			result.ContextAttachments = questionReferenceAttachments(saved)
			if err := e.validateAttachments(result.ContextAttachments); err != nil {
				return err
			}
			if err := e.addRequirement(saved, &result); err != nil {
				return err
			}
		}
		saved.Messages = append(saved.Messages, result)
		return nil
	})
	return &result, err
}

func (e *Engine) updateMessage(id, messageID string, change func(*TaskMessage) error) error {
	_, err := e.Store.Update(id, func(saved *Task) error {
		for i := range saved.Messages {
			if saved.Messages[i].ID == messageID {
				if err := change(&saved.Messages[i]); err != nil {
					return err
				}
				saved.Messages[i].Updated = time.Now().UTC().Format(time.RFC3339Nano)
				return nil
			}
		}
		return errors.New("message_missing: message not found")
	})
	return err
}

func conversationContext(task *Task, messageID string) []byte {
	prior := make([]map[string]string, 0)
	recorded, requirements := 0, 0
	for _, message := range task.Messages {
		if message.ID == messageID {
			break
		}
		recorded++
		if message.Kind == messageRequirement {
			requirements++
		}
		if message.Status == "answered" {
			entry := map[string]string{"user": boundedActivityText(message.Content)}
			if message.Reply != nil {
				entry["agent"] = boundedActivityText(message.Reply.Summary)
			}
			prior = append(prior, entry)
		}
	}
	total := len(prior)
	if len(prior) > 8 {
		prior = prior[len(prior)-8:]
	}
	raw, _ := json.Marshal(map[string]any{"original_goal": taskGoal(task), "state": task.State, "round": task.Round, "project_report": task.Audit, "review": task.Review, "previous_dialogue": prior, "previous_batch": task.PreviousContext, "recorded_messages_total": recorded, "requirements_total": requirements, "dialogue_total": total, "dialogue_excerpted": total > len(prior) || recorded > len(prior)})
	return raw
}
