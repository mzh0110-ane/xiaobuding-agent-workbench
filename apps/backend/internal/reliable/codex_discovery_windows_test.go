//go:build windows

package reliable

import (
	"os"
	"syscall"
	"testing"
	"time"
)

type codexWindowsFileFixture struct {
	mode       os.FileMode
	attributes uint32
}

func (f codexWindowsFileFixture) Name() string       { return "codex.exe" }
func (f codexWindowsFileFixture) Size() int64        { return 512 }
func (f codexWindowsFileFixture) Mode() os.FileMode  { return f.mode }
func (f codexWindowsFileFixture) ModTime() time.Time { return time.Time{} }
func (f codexWindowsFileFixture) IsDir() bool        { return f.mode.IsDir() }
func (f codexWindowsFileFixture) Sys() any {
	return &syscall.Win32FileAttributeData{FileAttributes: f.attributes}
}

func TestCodexDiscoveryRejectsWindowsReparseAttributes(t *testing.T) {
	for _, mode := range []os.FileMode{0, os.ModeDir} {
		if codexFileRedirected(codexWindowsFileFixture{mode: mode}) {
			t.Fatal("ordinary native file metadata was rejected")
		}
		fixture := codexWindowsFileFixture{mode: mode, attributes: syscall.FILE_ATTRIBUTE_REPARSE_POINT}
		if !codexFileRedirected(fixture) {
			t.Fatal("Windows reparse metadata was accepted")
		}
	}
}
