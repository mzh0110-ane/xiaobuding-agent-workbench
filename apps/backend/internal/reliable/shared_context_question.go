package reliable

import (
	"context"
	"errors"
)

func sharedAttachmentReceipts(inputs []AttachmentInput) []Attachment {
	refs := make([]Attachment, 0, len(inputs))
	for _, input := range inputs {
		refs = append(refs, input.Attachment)
	}
	return refs
}

func (e *Engine) prepareQuestionContext(ctx context.Context, task *Task, message *TaskMessage) (*SharedContext, error) {
	shared, err := e.buildSharedContext(ctx, task, message.ID)
	if err != nil {
		return nil, err
	}
	if err = e.validateSharedContextReference(shared.Reference, shared.Reference.IndexSHA256, nil); err != nil {
		return nil, err
	}
	err = e.updateMessage(task.ID, message.ID, func(saved *TaskMessage) error {
		if saved.Status != messageReading {
			return errors.New("message delivery was interrupted")
		}
		saved.ContextHash = shared.Reference.IndexSHA256
		return nil
	})
	if err != nil {
		return nil, err
	}
	message.ContextHash = shared.Reference.IndexSHA256
	return shared, nil
}
