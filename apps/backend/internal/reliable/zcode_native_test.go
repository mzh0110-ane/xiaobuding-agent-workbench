package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZcodeConnectionReusesConfigurationAndRedactsSecrets(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "v2"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(home, "v2", "provider_config.json"), []byte(`{"apiKey":"fixture-private-key"}`), 0600))
	executable := filepath.Join(home, "native.exe")
	require.NoError(t, os.WriteFile(executable, []byte("fixture"), 0600))
	c := ConnectionSettings{ID: "zcode", Kind: "zcode", Home: home, Provider: "native-provider", Model: "native-model", Entry: executable, Executable: executable, Models: []ModelOption{{ID: "native-model", Name: "Native", Efforts: []string{"high"}}}}
	providers, router, redact, err := buildConnections(Settings{Connections: []ConnectionSettings{c}})
	require.NoError(t, err)
	require.True(t, providers[0].Available)
	require.Equal(t, "zcode", providers[0].ID)
	require.IsType(t, &ZcodeRunner{}, router.Runners["zcode"])
	require.NotContains(t, string(redact([]byte("fixture-private-key"))), "fixture-private-key")
	c.Entry = "relative.cjs"
	_, _, _, err = buildConnections(Settings{Connections: []ConnectionSettings{c}})
	require.Error(t, err)
}

func TestZcodeNativeImplementationReport(t *testing.T) {
	if os.Getenv("XIAOBUDING_ZCODE_NATIVE_CHECK") != "1" {
		t.Skip("explicit native check not enabled")
	}
	root := os.Getenv("XIAOBUDING_ZCODE_CHECK_ROOT")
	require.NotEmpty(t, root)
	runner := &ZcodeRunner{Executable: os.Getenv("XIAOBUDING_ZCODE_EXECUTABLE"), Entry: os.Getenv("XIAOBUDING_ZCODE_ENTRY"), Home: os.Getenv("XIAOBUDING_ZCODE_HOME"), Provider: os.Getenv("XIAOBUDING_ZCODE_PROVIDER"), Model: os.Getenv("XIAOBUDING_ZCODE_MODEL")}
	result, err := runner.Run(context.Background(), RoleRequest{Role: roleImplementer, Directory: filepath.Join(root, "sample"), OutDir: filepath.Join(root, "native-go-proof"), ReadOnly: false, Effort: "high", Prompt: "Connection check only. Do not use tools or read/write files. Return a minimal valid JSON matching the provided schema, with no claimed inspections."})
	require.NoError(t, err)
	require.NotEmpty(t, result.Session)
	require.True(t, json.Valid(result.Output))
}

func TestZcodeIndependentReviewRemainsProtected(t *testing.T) {
	e := &Engine{}
	require.ErrorContains(t, e.validatePlan(ModelPlan{Reviewer: connectionZcode}), "review")
}
