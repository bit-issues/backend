package webhooks

import "github.com/bit-issues/backend/internal/webhooks/management"

// WebhookStatusResponse implements the canonical webhook contract success
// body. Status is one of registered, not_registered, failed, unavailable.
type WebhookStatusResponse struct {
	Status        string  `json:"status"         example:"registered"`
	HookUUID      *string `json:"hook_uuid"      example:"{abc-123}"`
	CallbackURL   string  `json:"callback_url"   example:"https://bitissues.example.com/api/v1/webhooks/bitbucket/push"`
	FailureReason *string `json:"failure_reason" example:"Bitbucket OAuth is not connected"`
	UpdatedAt     *string `json:"updated_at"     example:"2026-09-29T12:00:00Z"`
}

// NewWebhookStatusResponse maps a management status to the wire DTO.
func NewWebhookStatusResponse(st *management.WebhookStatus) WebhookStatusResponse {
	if st == nil {
		return WebhookStatusResponse{}
	}

	return WebhookStatusResponse{
		Status:        string(st.Status),
		HookUUID:      st.HookUUID,
		CallbackURL:   st.CallbackURL,
		FailureReason: st.FailureReason,
		UpdatedAt:     st.UpdatedAt,
	}
}
