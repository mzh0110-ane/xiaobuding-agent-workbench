package reliable

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReadOnlyNativePresetContinuesIntoMatchingWriterAndCodexReview(t *testing.T) {
	for _, id := range []string{connectionCodex, connectionHarness, connectionZcode} {
		t.Run(id, func(t *testing.T) {
			e, parent := fixtureEngine(t, &fixtureAgent{}, passingCheck)
			e.Providers = append(testProviders(), Provider{ID: connectionZcode, ToolKind: connectionZcode, Available: true, Model: "flash", Models: []ModelOption{{ID: "flash", Efforts: []string{"high"}}}})
			provider, err := e.provider(id)
			require.NoError(t, err)
			p := ModelPlan{AssistantMode: id, Mode: modeFullstack, Auditor: id, AuditorModel: provider.Model, Reviewer: connectionCodex, Frontend: connectionHarness, Backend: connectionCodex}
			if id == connectionZcode {
				p.AuditorEffort = "high"
			}
			_, err = e.Store.Update(parent.ID, func(task *Task) error {
				task.ReadOnly = true
				task.State = stateInspected
				task.ModelPlan = p
				return nil
			})
			require.NoError(t, err)
			message, err := e.startLinkedBatch(parent.ID, messageRequest{ID: "continue", Kind: messageRequirement, Content: "Now update app.txt"})
			require.NoError(t, err)
			child, err := e.Store.Get(message.LinkedTask)
			require.NoError(t, err)
			require.False(t, child.ReadOnly)
			require.Equal(t, parent.Project, child.Project)
			require.Equal(t, connectionCodex, child.ModelPlan.Reviewer)
			for _, domain := range []string{domainFrontend, domainBackend} {
				writer, model, effort := child.ModelPlan.selection(roleImplementer, domain)
				require.Equal(t, id, writer)
				require.Equal(t, provider.Model, model)
				require.Equal(t, p.AuditorEffort, effort)
			}
			if id == connectionZcode {
				require.Equal(t, connectionCodex, child.ModelPlan.Auditor)
			} else {
				require.Equal(t, id, child.ModelPlan.Auditor)
			}
			require.NoError(t, e.validatePlan(child.ModelPlan))
			saved, err := e.Store.Get(parent.ID)
			require.NoError(t, err)
			require.Equal(t, p, saved.ModelPlan)
			require.Zero(t, saved.Round)
		})
	}
}
