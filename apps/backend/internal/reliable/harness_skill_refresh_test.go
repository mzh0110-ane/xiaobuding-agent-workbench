package reliable

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHarnessExtensionChangesAreReadOnEveryInvocation(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node runtime unavailable")
	}
	home, entry := nativeInheritanceFixture(t)
	runner := &HarnessRunner{Executable: node, Entry: entry, Home: home}
	profile := filepath.Join(home, "profiles", "desktop", "cordis.patch.yml")
	read := func() (string, error) {
		patch, cleanup, loadErr := runner.inheritedExtensions(context.Background(), RoleRequest{Directory: t.TempDir(), OutDir: t.TempDir()}, append(os.Environ(), "DSH_HOME="+home))
		defer cleanup()
		if loadErr != nil {
			return "", loadErr
		}
		raw, readErr := os.ReadFile(patch)
		return string(raw), readErr
	}
	require.NoError(t, os.WriteFile(profile, []byte(`[{"id":"native-skill","name":"installed-skill","disabled":true,"config":{"freshness":"version-one"}}]`), 0600))
	first, err := read()
	require.NoError(t, err)
	require.Contains(t, first, "version-one")
	require.NoError(t, os.WriteFile(profile, []byte(`[{"id":"native-skill","name":"installed-skill","disabled":false,"config":{"freshness":"version-two"}}]`), 0600))
	second, err := read()
	require.NoError(t, err)
	require.Contains(t, second, "version-two")
	require.NotContains(t, second, "version-one")
	require.Contains(t, second, `"disabled":false`)
	require.NoError(t, os.Remove(profile))
	_, err = read()
	require.Error(t, err, "Missing current configuration must not fall back to a stale extension snapshot")
}
