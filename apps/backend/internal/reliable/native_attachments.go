package reliable

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const nativeImageByteLimit int64 = 20 << 20
const nativeImageCountLimit = 8

//go:embed native-attachments.mjs
var nativeAttachmentBridge []byte

// Native initial inputs stay bounded independently of the lifetime file history.
// Every image omitted here remains an ordinary, explicitly unread file reference.
func nativeImageInputs(attachments []AttachmentInput) (images, references []AttachmentInput) {
	selected := make(map[int]bool)
	var total int64
	for i := len(attachments) - 1; i >= 0; i-- {
		item := attachments[i]
		if item.Image && nativeImageMediaType(item.MediaType) && item.Size > 0 && item.Size <= nativeImageByteLimit-total && len(selected) < nativeImageCountLimit {
			selected[i], total = true, total+item.Size
		}
	}
	for i, item := range attachments {
		if selected[i] {
			images = append(images, item)
		} else if item.Image {
			references = append(references, item)
		}
	}
	return images, references
}

func nativeImageMediaType(value string) bool {
	switch value {
	case attachmentPNGMediaType, "image/jpeg", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func nativeAttachmentPrompt(req RoleRequest) string {
	if len(req.Attachments) == 0 {
		return req.Prompt
	}
	images, references := nativeImageInputs(req.Attachments)
	var out strings.Builder
	out.WriteString(req.Prompt)
	out.WriteString("\nNATIVE ATTACHMENT DELIVERY:\nCandidate native initial image inputs (Harness confirms actual delivery in its injected message):\n")
	for _, image := range images {
		fmt.Fprintf(&out, "- %q\n", image.Path)
	}
	out.WriteString("Image file references not attached as native images (older, larger, or unsupported formats):\n")
	for _, reference := range references {
		fmt.Fprintf(&out, "- %q\n", reference.Path)
	}
	out.WriteString("Use native image tools to inspect referenced images when supported. If the current route cannot inspect images, explicitly report that limitation; do not describe unseen content. A filename, a path, a successful upload, or a delivery receipt is not proof of understanding. General documents remain file references and require actual native tools or existing installed skills to read them.\n")
	return out.String()
}

func validateNativeImageInputs(attachments []AttachmentInput) error {
	images, _ := nativeImageInputs(attachments)
	for _, image := range images {
		if err := validateNativeImageFile(image); err != nil {
			return fmt.Errorf("attachment_changed: native image %q is unavailable or changed: %w", image.Name, err)
		}
	}
	return nil
}

func validateNativeImageFile(image AttachmentInput) error {
	if !filepath.IsAbs(image.Path) {
		return errors.New("image path must be absolute")
	}
	info, err := os.Lstat(image.Path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != image.Size {
		return errors.New("image must be the original regular file")
	}
	file, err := os.Open(image.Path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("image identity changed before opening")
	}
	hash := sha256.New()
	length, err := io.Copy(hash, io.LimitReader(file, nativeImageByteLimit+1))
	if err != nil || length != image.Size || hex.EncodeToString(hash.Sum(nil)) != image.SHA256 {
		return errors.New("image bytes no longer match their upload receipt")
	}
	return nil
}

func (r *HarnessRunner) nativeImagePatch(req RoleRequest) (string, error) {
	images, _ := nativeImageInputs(req.Attachments)
	if len(images) == 0 {
		return "", nil
	}
	if err := validateNativeImageInputs(req.Attachments); err != nil {
		return "", err
	}
	module := filepath.Join(req.OutDir, "native-attachments.mjs")
	if err := os.WriteFile(module, nativeAttachmentBridge, 0600); err != nil {
		return "", err
	}
	config := map[string]any{"entry": r.Entry, "images": images, "receipt": filepath.Join(req.OutDir, "native-image-delivery.json")}
	patch := []map[string]any{{"insert": []map[string]any{{"id": "xiaobuding-native-images", "name": nativeFileModuleURL(module), "config": config}}}}
	raw, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(req.OutDir, "native-image-input.yaml")
	return path, os.WriteFile(path, raw, 0600)
}

func nativeFileModuleURL(path string) string {
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}
