package reliable

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttachmentReportCitationsCannotReplaceInvalidProjectSources(t *testing.T) {
	ref := Attachment{ID: "supplied-input", SHA256: "verified-content"}
	for _, cited := range []string{"supplied-input", "not-supplied"} {
		t.Run(cited, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"summary": "Attachment report", "read_files": []string{"../outside.txt"}, "read_attachments": []string{cited}})
			require.NoError(t, err)
			require.NotPanics(t, func() {
				_, _, reportErr := decodeInspectionReport(t.TempDir(), &AgentResult{Session: "citation", Output: raw}, []Attachment{ref})
				require.Error(t, reportErr)
			})
		})
	}
}
