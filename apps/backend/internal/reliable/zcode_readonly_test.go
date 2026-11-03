package reliable

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestZcodeReadOnlyReportRequiresNativePermissionProof(t *testing.T) {
	success := `{"type":"zcode.event","event":{"type":"turn.completed","sessionId":"s","payload":{"resultType":"success"}}}` + "\n" + `{"type":"result","session":"s","response":"{}"}` + "\n"
	_, err := parseZcodeResult([]byte(success), true)
	require.ErrorContains(t, err, "permission proof")
	proof := `{"type":"zcode.event","event":{"type":"native_readonly_confirmed","session":"s","planEnabled":true}}` + "\n"
	_, err = parseZcodeResult([]byte(proof+success), true)
	require.NoError(t, err)
}

func TestZcodeNativeReadOnlyInspectsSourceWithoutWrites(t *testing.T) {
	if os.Getenv("XIAOBUDING_ZCODE_READONLY_CHECK") != "1" {
		t.Skip("explicit native read-only check disabled")
	}
	project := t.TempDir()
	out := t.TempDir()
	marker := "readonly_" + randomID()
	t.Cleanup(func() {
		if destination := os.Getenv("XIAOBUDING_ZCODE_READONLY_PROOF"); destination != "" {
			for _, name := range []string{"events.jsonl", "stderr.txt", "response.json"} {
				if raw, err := os.ReadFile(filepath.Join(out, name)); err == nil {
					_ = os.WriteFile(destination+"."+name, raw, 0600)
				}
			}
		}
	})
	require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte(marker), 0600))
	r := &ZcodeRunner{Executable: os.Getenv("XIAOBUDING_ZCODE_EXECUTABLE"), Entry: os.Getenv("XIAOBUDING_ZCODE_ENTRY"), Home: os.Getenv("XIAOBUDING_ZCODE_HOME"), Provider: os.Getenv("XIAOBUDING_ZCODE_PROVIDER"), Model: os.Getenv("XIAOBUDING_ZCODE_MODEL")}
	result, err := r.Run(context.Background(), RoleRequest{Role: roleAuditor, ReadOnly: true, Directory: project, OutDir: out, Effort: "high", Prompt: "Read app.txt with the native Read tool. Return a valid report with its exact contents in summary and the project-relative string app.txt (not an absolute path) in read_files. Do not edit any file, execute commands, install anything or leave planning mode."})
	require.NoError(t, err)
	if destination := os.Getenv("XIAOBUDING_ZCODE_READONLY_PROOF"); destination != "" {
		events, _ := os.ReadFile(filepath.Join(out, "events.jsonl"))
		_ = os.WriteFile(destination, events, 0600)
		_ = os.WriteFile(destination+".response.json", result.Output, 0600)
	}
	var report Audit
	require.NoError(t, json.Unmarshal(result.Output, &report))
	_, _, decodeErr := decodeInspectionReport(project, result)
	require.NoError(t, decodeErr)
	require.Contains(t, report.Summary, marker)
	require.Contains(t, report.ReadFiles, "app.txt")
	raw, err := os.ReadFile(filepath.Join(project, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, marker, string(raw))
	files, err := os.ReadDir(project)
	require.NoError(t, err)
	require.Len(t, files, 1)
	if destination := os.Getenv("XIAOBUDING_ZCODE_READONLY_PROOF"); destination != "" {
		events, readErr := os.ReadFile(filepath.Join(out, "events.jsonl"))
		require.NoError(t, readErr)
		require.NoError(t, os.WriteFile(destination, events, 0600))
	}
}
