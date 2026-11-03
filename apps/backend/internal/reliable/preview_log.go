package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type previewReady struct {
	Port string
}

type previewLog struct {
	mu     sync.Mutex
	file   *os.File
	tail   []byte
	line   []byte
	key    string
	ready  chan previewReady
	cancel context.CancelFunc
	redact func([]byte) []byte
	total  int
	closed bool
}

func newPreviewLog(engine *Engine, id, key string, cancel context.CancelFunc) (*previewLog, error) {
	root, err := openStorageRoot(engine.Root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	if err = root.MkdirAll(filepath.Join("cache", "previews"), 0700); err != nil {
		return nil, err
	}
	name := filepath.Join("cache", "previews", id+".log")
	if err = validatePreviewLogDirectory(root); err != nil {
		return nil, err
	}
	file, err := openPreviewLogFile(root, name)
	if err != nil {
		return nil, err
	}
	return &previewLog{file: file, key: key, ready: make(chan previewReady, 1), cancel: cancel, redact: engine.Redact}, nil
}

func validatePreviewLogDirectory(root *os.Root) error {
	for _, part := range []string{"cache", filepath.Join("cache", "previews")} {
		info, statErr := root.Lstat(part)
		if statErr != nil || !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return errors.New("preview_log_unsafe: log directory is redirected")
		}
	}
	return nil
}

func openPreviewLogFile(root *os.Root, name string) (*os.File, error) {
	before, statErr := root.Lstat(name)
	if statErr != nil && !os.IsNotExist(statErr) || statErr == nil && !before.Mode().IsRegular() {
		return nil, errors.New("preview_log_unsafe: log path is not a regular file")
	}
	file, err := root.OpenFile(name, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || before != nil && !os.SameFile(before, opened) {
		_ = file.Close()
		return nil, errors.New("preview_log_unsafe: log file identity changed")
	}
	if err = file.Truncate(0); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func (log *previewLog) Write(raw []byte) (int, error) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.total += len(raw)
	if log.total > 2<<20 {
		log.cancel()
		return 0, errors.New("preview_log_limit: runtime output exceeded 2 MiB")
	}
	log.line = append(log.line, raw...)
	for {
		index := bytes.IndexByte(log.line, '\n')
		if index < 0 {
			break
		}
		line := log.line[:index+1]
		if !log.captureReady(line) {
			log.tail = append(log.tail, line...)
		}
		log.line = log.line[index+1:]
	}
	if len(log.line) > 32<<10 {
		log.tail = append(log.tail, log.line[:len(log.line)-(16<<10)]...)
		log.line = log.line[len(log.line)-(16<<10):]
	}
	if log.redact != nil {
		log.tail = log.redact(log.tail)
	}
	if len(log.tail) > 16<<10 {
		log.tail = log.tail[len(log.tail)-(16<<10):]
	}
	if _, err := log.file.Seek(0, 0); err != nil {
		return 0, err
	}
	if err := log.file.Truncate(0); err != nil {
		return 0, err
	}
	_, err := log.file.Write(log.tail)
	return len(raw), err
}

func (log *previewLog) captureReady(line []byte) bool {
	var event struct {
		Key  string `json:"xbd_preview_ready"`
		Port string `json:"port"`
	}
	if json.Unmarshal(line, &event) != nil || event.Key != log.key {
		return false
	}
	port, err := strconv.Atoi(event.Port)
	if err != nil || port < 1 || port > 65535 {
		return true
	}
	select {
	case log.ready <- previewReady{Port: event.Port}:
	default:
	}
	return true
}

func (log *previewLog) Tail() string {
	log.mu.Lock()
	defer log.mu.Unlock()
	return strings.ToValidUTF8(string(log.tail), "�")
}

func (log *previewLog) Close() error {
	log.mu.Lock()
	defer log.mu.Unlock()
	if log.closed {
		return nil
	}
	log.closed = true
	return log.file.Close()
}
