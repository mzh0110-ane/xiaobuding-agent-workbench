package reliable

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestZcodePresetUsesCodexAuditAndReview(t *testing.T) {
	e := &Engine{Providers: append(testProviders(), Provider{ID: connectionZcode, ToolKind: connectionZcode, Available: true, Model: "zcode-flash", Models: []ModelOption{{ID: "zcode-flash", Efforts: []string{"high", "max"}}}})}
	for _, mode := range []string{"", domainFrontend, domainBackend, modeFullstack} {
		p := ModelPlan{AssistantMode: connectionZcode, Mode: mode, Auditor: connectionCodex, Reviewer: connectionCodex, Implementer: connectionZcode, ImplementerModel: "zcode-flash", Frontend: connectionZcode, FrontendModel: "zcode-flash", Backend: connectionZcode, BackendModel: "zcode-flash"}
		require.NoError(t, e.validatePlan(p))
		require.Error(t, e.validateTaskPlan(p, true))
		changed := p
		changed.Auditor = connectionHarness
		require.Error(t, e.validatePlan(changed))
		changed = p
		changed.Reviewer = connectionHarness
		require.Error(t, e.validatePlan(changed))
		if mode == modeFullstack {
			changed = p
			changed.Backend = connectionCodex
			require.Error(t, e.validatePlan(changed))
		}
	}
}

func TestZcodeReadOnlyPresetUsesNativePlanner(t *testing.T) {
	e := &Engine{Providers: append(testProviders(), Provider{ID: connectionZcode, ToolKind: connectionZcode, Available: true, Model: "flash", Models: []ModelOption{{ID: "flash", Efforts: []string{"high"}}}})}
	p := ModelPlan{AssistantMode: connectionZcode, Auditor: connectionZcode, AuditorModel: "flash", AuditorEffort: "high"}
	require.NoError(t, e.validateTaskPlan(p, true))
	require.Error(t, e.validateTaskPlan(p, false))
}
