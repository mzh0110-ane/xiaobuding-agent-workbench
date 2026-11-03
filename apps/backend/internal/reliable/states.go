package reliable

const stateInspected = "inspected"

const (
	stateQueued             = "queued"
	statePreparing          = "preparing"
	stateAuditing           = "auditing"
	stateImplementing       = "implementing"
	stateChecking           = "checking"
	stateReviewing          = "reviewing"
	stateAwaitingAcceptance = "awaiting_acceptance"
	stateAccepted           = "accepted"
	stateBlocked            = "blocked"
	stateInterrupted        = "interrupted"
	stateExhausted          = "exhausted"
	stateCancelling         = "cancelling"
	stateCancelled          = "cancelled"
	stateApplying           = "applying"
	roleAuditor             = "auditor"
	roleImplementer         = "implementer"
	roleReviewer            = "reviewer"
)
