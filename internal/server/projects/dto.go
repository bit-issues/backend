package projects

import (
	"github.com/bit-issues/backend/internal/projects"
	"github.com/bit-issues/backend/internal/server/dto"
	clientgo "github.com/bit-issues/client-go"
	"github.com/samber/lo"
)

func NewProjectListResponse(items []projects.Project, total int) clientgo.ProjectListResponse {
	return clientgo.ProjectListResponse{
		Items: lo.Map(
			items,
			func(p projects.Project, _ int) clientgo.Project {
				return *dto.ToProject(&p)
			},
		),
		Total: total,
	}
}

// Conversion functions

type projectRequest clientgo.ProjectRequest

// toProjectInput converts the shared project request to a domain ProjectInput.
func (req *projectRequest) toProjectInput() projects.ProjectInput {
	return projects.ProjectInput{
		Name:    req.Name,
		RepoURL: req.RepoURL,
	}
}

type projectUpdateRequest clientgo.ProjectUpdateRequest

// toProjectUpdate converts the shared project update request to a domain ProjectUpdate.
func (req *projectUpdateRequest) toProjectUpdate() projects.ProjectUpdate {
	return projects.ProjectUpdate{
		Name:    req.Name,
		RepoURL: req.RepoURL,
	}
}
