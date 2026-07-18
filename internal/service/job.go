package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/repository"
)

// JobService handles job posting logic and ownership checks.
type JobService struct {
	jobs *repository.JobRepo
}

// NewJobService builds a JobService.
func NewJobService(jobs *repository.JobRepo) *JobService {
	return &JobService{jobs: jobs}
}

// Create validates and inserts a job owned by creatorID.
func (s *JobService) Create(ctx context.Context, creatorID int64, j *domain.Job) error {
	if strings.TrimSpace(j.Title) == "" || strings.TrimSpace(j.Description) == "" {
		return fmt.Errorf("%w: title and description are required", domain.ErrValidation)
	}
	j.CreatedBy = creatorID
	if j.Status == "" {
		j.Status = domain.JobOpen
	}
	return s.jobs.Create(ctx, j)
}

// Get returns a single job.
func (s *JobService) Get(ctx context.Context, id int64) (*domain.Job, error) {
	return s.jobs.GetByID(ctx, id)
}

// List returns jobs filtered by status and search text.
func (s *JobService) List(ctx context.Context, status domain.JobStatus, search string, limit, offset int) ([]domain.Job, error) {
	return s.jobs.List(ctx, status, search, limit, offset)
}

// Update modifies a job. Only the owning recruiter or an admin may edit it.
func (s *JobService) Update(ctx context.Context, actorID int64, role domain.Role, j *domain.Job) error {
	existing, err := s.jobs.GetByID(ctx, j.ID)
	if err != nil {
		return err
	}
	if role != domain.RoleAdmin && existing.CreatedBy != actorID {
		return domain.ErrForbidden
	}
	j.CreatedBy = existing.CreatedBy // creator is immutable
	if j.Status == "" {
		j.Status = existing.Status
	}
	return s.jobs.Update(ctx, j)
}

// Close marks a job closed. Only the owning recruiter or an admin may close it.
func (s *JobService) Close(ctx context.Context, actorID int64, role domain.Role, id int64) error {
	existing, err := s.jobs.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role != domain.RoleAdmin && existing.CreatedBy != actorID {
		return domain.ErrForbidden
	}
	return s.jobs.SetStatus(ctx, id, domain.JobClosed)
}
