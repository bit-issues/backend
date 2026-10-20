package webhooks

// ClassifyWebhookErrorForTest exposes classifyWebhookError to the external
// test package. This file is only compiled during testing, so it does not
// expand the production API.
func ClassifyWebhookErrorForTest(err error) error {
	return classifyWebhookError(err)
}
