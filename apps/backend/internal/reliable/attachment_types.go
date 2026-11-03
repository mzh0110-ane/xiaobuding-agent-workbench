package reliable

const attachmentPNGMediaType = "image/png"

type Attachment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	SHA256    string `json:"sha256"`
	Created   string `json:"created"`
	Size      int64  `json:"size"`
	Image     bool   `json:"image"`
}

type AttachmentInput struct {
	Attachment
	Path string `json:"path"`
}
