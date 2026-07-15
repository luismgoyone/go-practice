package service

import (
	"context"
	"strings"

	"github.com/luismgoyone/go-practice/internal/model"
	"github.com/luismgoyone/go-practice/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectService struct {
	repo *repository.ProjectRepo
}

func NewProjectService(repo *repository.ProjectRepo) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) Create(ctx context.Context, name, description string) (model.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Project{}, &ValidationError{Message: "name is required"}
	}
	return s.repo.Create(ctx, model.Project{
		Name:        name,
		Description: description,
	})
}

func (s *ProjectService) List(ctx context.Context) ([]model.Project, error) {
	return s.repo.List(ctx)
}

func (s *ProjectService) GetByID(ctx context.Context, id primitive.ObjectID) (model.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) Update(ctx context.Context, id primitive.ObjectID, name, description *string) (model.Project, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return model.Project{}, err
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return model.Project{}, &ValidationError{Message: "name is required"}
		}
		p.Name = n
	}
	if description != nil {
		p.Description = *description
	}
	return s.repo.Update(ctx, p)
}

func (s *ProjectService) Delete(ctx context.Context, id primitive.ObjectID) error {
	return s.repo.Delete(ctx, id)
}
