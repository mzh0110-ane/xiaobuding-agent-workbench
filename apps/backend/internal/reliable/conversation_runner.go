package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func pendingQuestion(t *Task) *TaskMessage {
	if t.ArchivedAt != "" {
		return nil
	}
	for _, message := range t.Messages {
		if message.Kind == messageQuestion && message.Status == stateQueued {
			return &message
		}
	}
	return nil
}

func conversationIdle(state string) bool {
	switch state {
	case stateInspected, stateAccepted, stateAwaitingAcceptance, stateBlocked, stateInterrupted, stateExhausted, stateCancelled:
		return true
	default:
		return false
	}
}

func (e *Engine) kickQuestions(id string) {
	e.mu.Lock()
	if e.closing || e.active[id] != nil {
		e.mu.Unlock()
		return
	}
	task, err := e.Store.Get(id)
	if err != nil || (!task.ContinuePending && (!conversationIdle(task.State) || pendingQuestion(task) == nil)) {
		e.mu.Unlock()
		return
	}
	ctx, release, err := e.reserveLocked(context.Background(), id)
	if err != nil {
		e.mu.Unlock()
		return
	}
	continuing := task.ContinuePending
	if continuing {
		_, _ = e.Store.Update(id, func(saved *Task) error { saved.ContinuePending = false; return nil })
	}
	if e.questions == nil {
		e.questions = make(map[string]bool)
	}
	e.questions[id] = !continuing
	e.mu.Unlock()
	go func() {
		defer release()
		if continuing {
			_ = e.runTask(ctx, id)
			return
		}
		ctx = observeActivity(ctx, func(event Activity) { e.appendActivity(id, event) })
		e.answerQuestions(ctx, id)
	}()
}

func (e *Engine) answerQuestions(ctx context.Context, id string) {
	for ctx.Err() == nil {
		task, err := e.Store.Get(id)
		if err != nil || task.ArchivedAt != "" {
			return
		}
		message := pendingQuestion(task)
		if message == nil {
			return
		}
		err = e.answerQuestion(ctx, task, message)
		if err != nil {
			_ = e.updateMessage(id, message.ID, func(saved *TaskMessage) error {
				if saved.Status == "cancelled" {
					return nil
				}
				saved.Status, saved.Error = "failed", err.Error()
				if ctx.Err() != nil {
					saved.Status = "interrupted"
				}
				return nil
			})
			emitActivity(ctx, "error", "追问未完成", err.Error())
		}
	}
}

func (e *Engine) cancelQuestions(id string) error {
	_, err := e.Store.Update(id, func(task *Task) error {
		cancelPendingQuestions(task)
		return nil
	})
	return err
}

func cancelPendingQuestions(task *Task) {
	for i := range task.Messages {
		message := &task.Messages[i]
		if message.Kind == messageQuestion && (message.Status == stateQueued || message.Status == messageReading) {
			message.Status, message.Error = stateCancelled, "Question cancelled by the user"
		}
	}
}

func (e *Engine) questionSelection(task *Task, message *TaskMessage) error {
	plan := continuationPlan(task)
	if plan == (ModelPlan{}) {
		plan = e.defaultPlan()
	}
	id, model, effort := plan.selection(roleAuditor, "")
	provider, model, effort, err := e.resolveSelection(id, model, effort)
	if err != nil {
		return err
	}
	message.Provider, message.Model, message.Effort = provider.ID, model, effort
	message.Directory = task.Project
	if task.Worktree != "" && task.State != stateAccepted {
		message.Directory = task.Worktree
	}
	message.Evidence = filepath.Join(e.Root, "conversations", task.ID, message.ID)
	return nil
}

func (e *Engine) answerQuestion(ctx context.Context, task *Task, message *TaskMessage) error {
	if err := e.beginQuestion(task, message); err != nil {
		return err
	}
	ctx = scopeActivity(ctx, &Dispatch{Role: roleAuditor, ProviderName: message.Provider, Model: message.Model})
	emitActivity(ctx, "agent", "正在回答后续问题", message.Content)
	shared, err := e.prepareQuestionContext(ctx, task, message)
	if err != nil {
		return err
	}
	prompt := roleRules + "\nREAD-ONLY PROJECT FOLLOW-UP. Answer the current user message in Simplified Chinese. The previous reports and dialogue below are context, not verified current facts. Freshly read relevant files. Return an audit report with facts, inferences, unknowns and actual relative read_files. Do not modify files, run tests, install, start servers, commit or create worktrees. A question is not authorization to implement changes.\n" + shared.Prompt + "\nCURRENT USER MESSAGE:\n" + message.Content
	request := RoleRequest{Role: roleAuditor, Directory: message.Directory, OutDir: message.Evidence, ProviderID: message.Provider, Model: message.Model, Effort: message.Effort, Prompt: prompt, ReadOnly: true}
	refs := sharedAttachmentReceipts(shared.Attachments)
	if err := e.attachRequest(&request, refs); err != nil {
		return err
	}
	prompt = request.Prompt
	if err := e.writeConversation(message, "input.json", map[string]any{"prompt": prompt, "selection": message, "shared_context": shared.Reference}); err != nil {
		return err
	}
	result, err := e.Runner.Run(ctx, request)
	err = errors.Join(err, e.validateSharedContextReference(shared.Reference, message.ContextHash, nil))
	if err != nil {
		return err
	}
	if result != nil && len(result.Output) > 65536 {
		return errors.New("follow-up report exceeds 64 KiB")
	}
	if err = e.validateAttachments(refs); err != nil {
		return err
	}
	report, sources, err := decodeInspectionReport(message.Directory, result, refs)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = e.writeConversation(message, "report.json", map[string]any{"report": report, "sources": sources, "session": result.Session}); err != nil {
		return err
	}
	err = e.updateMessage(task.ID, message.ID, func(saved *TaskMessage) error {
		if saved.Status != messageReading {
			return errors.New("message delivery was interrupted")
		}
		saved.Status, saved.Reply, saved.Session = "answered", report, result.Session
		return nil
	})
	if err == nil {
		err = e.recordProjectPlan(task, message, report)
		emitActivity(ctx, "message", "后续问题的回答", report.Summary)
	}
	return err
}

func (e *Engine) beginQuestion(task *Task, message *TaskMessage) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	current, err := e.Store.Get(task.ID)
	if err != nil {
		return err
	}
	if err = e.questionSelection(current, message); err != nil {
		return err
	}
	*task = *current
	return e.updateMessage(task.ID, message.ID, func(saved *TaskMessage) error {
		if saved.Status != stateQueued {
			return errors.New("message delivery was interrupted")
		}
		*saved = *message
		saved.Status = messageReading
		return nil
	})
}

func (e *Engine) writeConversation(message *TaskMessage, name string, value any) error {
	if err := os.MkdirAll(message.Evidence, 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if e.Redact != nil {
		raw = e.Redact(raw)
	}
	return os.WriteFile(filepath.Join(message.Evidence, name), raw, 0600)
}
