package reliable

import "errors"

func inspectionAttachmentSources(refs []Attachment, read []string) (map[string]string, error) {
	if len(read) > 100 {
		return nil, errors.New("attachment_report: at most 100 attachment citations are allowed")
	}
	known := make(map[string]string, len(refs))
	for _, ref := range refs {
		known[ref.ID] = ref.SHA256
	}
	sources := make(map[string]string, len(read))
	for _, id := range read {
		hash, ok := known[id]
		if !ok || hash == "" {
			return nil, errors.New("attachment_report: an attachment citation was not supplied to this call")
		}
		sources["attachment/"+id] = hash
	}
	return sources, nil
}
