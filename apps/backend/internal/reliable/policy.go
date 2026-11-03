package reliable

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type Finding struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Reason   string `json:"reason"`
}

type Review struct {
	Verdict   string    `json:"verdict"`
	Summary   string    `json:"summary"`
	ReadFiles []string  `json:"read_files"`
	Findings  []Finding `json:"findings"`
}

type Decision string

const (
	Pass      Decision = "pass"
	Rework    Decision = "rework"
	Block     Decision = stateBlocked
	MaxRounds          = 3
)

func Decide(raw json.RawMessage, checksPassed, snapshotMatches bool, required []string) (Decision, string) {
	if !snapshotMatches {
		return Block, "Source changed during verification or review"
	}
	var review Review
	if err := decodeStrict(raw, &review); err != nil {
		return Block, "Invalid review: " + err.Error()
	}
	if err := validateReviewShape(review, required); err != nil {
		return Block, err.Error()
	}
	serious, err := seriousFindings(review.Findings)
	if err != nil {
		return Block, err.Error()
	}
	if review.Verdict == stateBlocked {
		return Block, review.Summary
	}
	if !checksPassed || serious || review.Verdict == string(Rework) {
		return Rework, review.Summary
	}
	return Pass, review.Summary
}

func validateReviewShape(review Review, required []string) error {
	if strings.TrimSpace(review.Summary) == "" || review.Findings == nil || review.ReadFiles == nil {
		return errors.New("review is incomplete")
	}
	if review.Verdict != string(Pass) && review.Verdict != string(Rework) && review.Verdict != stateBlocked {
		return errors.New("review verdict is missing or unsupported")
	}
	read := make(map[string]bool)
	for _, path := range review.ReadFiles {
		read[path] = true
	}
	for _, path := range required {
		if !read[path] {
			return fmt.Errorf("review did not cover required file: %s", path)
		}
	}
	return nil
}

func seriousFindings(findings []Finding) (bool, error) {
	serious := false
	for _, f := range findings {
		if strings.TrimSpace(f.Reason) == "" {
			return false, errors.New("finding has no evidence or explanation")
		}
		switch f.Severity {
		case "P0", "P1":
			serious = true
		case "P2", "P3":
		default:
			return false, errors.New("unsupported finding severity")
		}
	}
	return serious, nil
}

func decodeStrict(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("unexpected trailing JSON")
	}
	return nil
}
