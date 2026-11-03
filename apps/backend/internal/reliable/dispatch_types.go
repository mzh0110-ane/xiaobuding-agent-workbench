package reliable

const dispatchNotStarted = "not_started"

type Dispatch struct {
	ContextHash   string `json:"context_hash,omitempty"`
	Domain        string `json:"domain,omitempty"`
	Effort        string `json:"effort,omitempty"`
	ID            string `json:"id"`
	Role          string `json:"role"`
	Round         int    `json:"round"`
	ProviderID    string `json:"provider_id"`
	ProviderName  string `json:"provider_name"`
	Model         string `json:"model"`
	PlanRevision  int    `json:"plan_revision"`
	SourceHash    string `json:"source_hash"`
	Session       string `json:"session"`
	Status        string `json:"status"`
	Started       string `json:"started"`
	Completed     string `json:"completed"`
	InputEvidence string `json:"input_evidence"`
	StartEvidence string `json:"start_evidence"`
	EndEvidence   string `json:"end_evidence"`
}
