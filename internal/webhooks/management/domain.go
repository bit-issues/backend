package management

// Status is the live management status of a project's Bitbucket webhook
// (canonical_webhook_contract).
type Status string

// Webhook status values (canonical_webhook_contract).
const (
	StatusRegistered    Status = "registered"
	StatusNotRegistered Status = "not_registered"
	StatusFailed        Status = "failed"
	StatusUnavailable   Status = "unavailable"
)

// WebhookStatus is the live management status of a project's Bitbucket
// webhook. No webhook state is persisted; every read queries Bitbucket.
type WebhookStatus struct {
	Status        Status
	HookUUID      *string
	CallbackURL   string
	FailureReason *string
	UpdatedAt     *string
}

// repoRef identifies a Bitbucket repository parsed from a project URL.
type repoRef struct {
	workspace string
	slug      string
}
