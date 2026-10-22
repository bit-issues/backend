package webhooks

import "github.com/bit-issues/backend/pkg/bitbucket"

// ValidateCallbackURLForTest exposes validateCallbackURL to the external test
// package. This file is only compiled during testing, so it does not expand
// the production API.
func ValidateCallbackURLForTest(raw string) (string, error) {
	return validateCallbackURL(raw)
}

// FindMatchingHookForTest exposes findMatchingHook to the external test package.
func FindMatchingHookForTest(hooks []bitbucket.Hook, webhookURL string) *bitbucket.Hook {
	return findMatchingHook(hooks, webhookURL)
}

// WebhookStatusFromHookForTest exposes webhookStatusFromHook to the external
// test package.
func WebhookStatusFromHookForTest(hook *bitbucket.Hook, webhookURL string) *WebhookStatus {
	return webhookStatusFromHook(hook, webhookURL)
}
