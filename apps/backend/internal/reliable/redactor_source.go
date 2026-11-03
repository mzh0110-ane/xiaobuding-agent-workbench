package reliable

import "sync"

type redactorSource struct {
	mu     sync.RWMutex
	filter func([]byte) []byte
}

func (s *redactorSource) snapshot() func([]byte) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.filter
}
func (s *redactorSource) replace(filter func([]byte) []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filter = filter
}
func (s *redactorSource) apply(raw []byte) []byte { return s.snapshot()(raw) }

func (s *redactorSource) bind(router *ProviderRouter) {
	for _, runner := range router.Runners {
		switch r := runner.(type) {
		case *CodexRunner:
			r.RedactorSource = s.snapshot
		case *HarnessRunner:
			r.RedactorSource = s.snapshot
		}
	}
}

func snapshotRedactor(base func([]byte) []byte, source func() func([]byte) []byte) func([]byte) []byte {
	if source != nil {
		base = source()
	}
	if base == nil {
		return func(raw []byte) []byte { return raw }
	}
	return base
}
