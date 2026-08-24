package projects

import (
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/server/dto"
	"github.com/bit-issues/backend/internal/webhooks"
	"github.com/samber/lo"
)

// ProjectRequest represents the request body for creating a new project.
type ProjectRequest struct {
	Name    string `json:"name"     validate:"required"     example:"Backend Service"`
	RepoURL string `json:"repo_url" validate:"required,url" example:"https://bitbucket.org/company/backend"`
}

// ProjectUpdateRequest represents the request body for updating a project.
// All fields are optional to support partial updates.
type ProjectUpdateRequest struct {
	Name    *string `json:"name,omitempty"     validate:"omitempty"     example:"Backend Service"`
	RepoURL *string `json:"repo_url,omitempty" validate:"omitempty,url" example:"https://bitbucket.org/company/backend"`
}

// ProjectResponse represents the API response for a single project.
type ProjectResponse struct {
	dto.Project
}

func NewProjectResponse(p *projects.Project) ProjectResponse {
	if p == nil {
		return ProjectResponse{}
	}

	return ProjectResponse{
		Project: *dto.ToProject(p),
	}
}

// ProjectListResponse represents the API response for a list of projects.
type ProjectListResponse struct {
	Items []ProjectResponse `json:"items"`
	Total int               `json:"total" example:"1"`
}

func NewProjectListResponse(items []projects.Project, total int) ProjectListResponse {
	return ProjectListResponse{
		Items: lo.Map(
			items,
			func(p projects.Project, _ int) ProjectResponse {
				return NewProjectResponse(&p)
			},
		),
		Total: total,
	}
}

// WebhookStatusResponse represents the live Bitbucket webhook state of a
// project. It never carries the webhook secret.
type WebhookStatusResponse struct {
	Status        string `json:"status"                   example:"registered"`
	HookUUID      string `json:"hook_uuid,omitempty"      example:"{abc-123}"`
	CallbackURL   string `json:"callback_url"             example:"https://issues.example.com/api/v1/webhooks/bitbucket/push"`
	FailureReason string `json:"failure_reason,omitempty" example:"insufficient Bitbucket permissions to manage webhooks"`
	UpdatedAt     string `json:"updated_at,omitempty"     example:"2026-08-20T10:00:00+00:00"`
}

func NewWebhookStatusResponse(s *webhooks.WebhookStatus) WebhookStatusResponse {
	if s == nil {
		return WebhookStatusResponse{}
	}

	return WebhookStatusResponse{
		Status:        s.Status,
		HookUUID:      s.HookUUID,
		CallbackURL:   s.CallbackURL,
		FailureReason: "",
		UpdatedAt:     s.UpdatedAt,
	}
}

// Conversion functions

// toProjectInput converts a ProjectRequest DTO to a domain ProjectInput.
func (req ProjectRequest) toProjectInput() projects.ProjectInput {
	return projects.ProjectInput{
		Name:    req.Name,
		RepoURL: req.RepoURL,
	}
}

// toProjectUpdate converts a ProjectUpdateRequest DTO to a domain ProjectUpdate.
func (req ProjectUpdateRequest) toProjectUpdate() projects.ProjectUpdate {
	return projects.ProjectUpdate{
		Name:    req.Name,
		RepoURL: req.RepoURL,
	}
}
