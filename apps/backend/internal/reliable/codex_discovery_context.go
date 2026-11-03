package reliable

import "context"

type codexExecutableContextKey struct{}

type codexExecutableSnapshot struct {
	path string
	err  error
}

func pinCodexExecutables(ctx context.Context, runner AgentRunner) context.Context {
	snapshots := make(map[*CodexRunner]codexExecutableSnapshot)
	add := func(codex *CodexRunner) {
		path, err := resolveCodexExecutable(codex.Executable)
		snapshots[codex] = codexExecutableSnapshot{path: path, err: err}
	}
	switch typed := runner.(type) {
	case *CodexRunner:
		add(typed)
	case *ProviderRouter:
		for _, connection := range typed.Runners {
			if codex, ok := connection.(*CodexRunner); ok {
				add(codex)
			}
		}
	}
	if len(snapshots) == 0 {
		return ctx
	}
	return context.WithValue(ctx, codexExecutableContextKey{}, snapshots)
}

func (r *CodexRunner) executableForContext(ctx context.Context) (string, error) {
	if snapshots, ok := ctx.Value(codexExecutableContextKey{}).(map[*CodexRunner]codexExecutableSnapshot); ok {
		if snapshot, pinned := snapshots[r]; pinned {
			return snapshot.path, snapshot.err
		}
	}
	return resolveCodexExecutable(r.Executable)
}
