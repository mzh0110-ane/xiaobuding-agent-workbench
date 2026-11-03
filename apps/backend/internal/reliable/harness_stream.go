package reliable

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

type harnessEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
	Phase     string `json:"phase"`
	Text      string `json:"text"`
	Reason    struct {
		Kind string `json:"kind"`
	} `json:"reason"`
}

type harnessStream struct {
	directory, session string
	completed, final   bool
	report             []byte
}

func parseHarnessEvents(raw []byte, dir string) (*AgentResult, error) {
	stream := harnessStream{directory: dir}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 16<<20)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var event harnessEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, errors.New("malformed native Harness event stream")
		}
		if err := stream.consume(event); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !stream.final || stream.session == "" {
		return nil, errors.New("native Harness did not finish an identified session")
	}
	return &AgentResult{Session: stream.session, Output: stream.report}, nil
}

func (s *harnessStream) consume(e harnessEvent) error {
	if s.final || e.Type == "" {
		return errors.New("native Harness emitted data after its final report or an unnamed event")
	}
	if s.session == "" && e.Type != "session" {
		return errors.New("native Harness did not establish its session first")
	}
	switch e.Type {
	case "session":
		if s.session != "" || e.SessionID == "" || projectKey(e.CWD) != projectKey(s.directory) {
			return errors.New("native Harness session is ambiguous or references another project")
		}
		s.session = e.SessionID
	case "status":
		switch e.Phase {
		case "turn_start":
			s.completed = false
		case "turn_end":
			s.completed = e.Reason.Kind == "completed"
		}
	case eventTypeError:
		return errors.New("native Harness reported an error; inspect its saved event log")
	case "final":
		return s.acceptFinal(e.Text)
	}
	return nil
}

func (s *harnessStream) acceptFinal(text string) error {
	raw := []byte(strings.TrimSpace(text))
	if !s.completed || len(raw) == 0 || raw[0] != '{' || !json.Valid(raw) {
		return errors.New("native Harness final report is incomplete or is not a JSON object")
	}
	s.final, s.report = true, raw
	return nil
}
