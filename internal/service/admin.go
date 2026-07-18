package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/faizan/ats/internal/auth"
	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/repository"
)

// AdminService handles recruiter provisioning and analytics.
type AdminService struct {
	users *repository.UserRepo
	jobs  *repository.JobRepo
	apps  *repository.ApplicationRepo
}

// NewAdminService builds an AdminService.
func NewAdminService(users *repository.UserRepo, jobs *repository.JobRepo, apps *repository.ApplicationRepo) *AdminService {
	return &AdminService{users: users, jobs: jobs, apps: apps}
}

// CreateRecruiter provisions a recruiter account.
func (s *AdminService) CreateRecruiter(ctx context.Context, email, password, fullName string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(password) < 8 || strings.TrimSpace(fullName) == "" {
		return nil, fmt.Errorf("%w: email, name, and an 8+ character password are required", domain.ErrValidation)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &domain.User{Email: email, FullName: fullName, Role: domain.RoleRecruiter, PasswordHash: hash}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// ListRecruiters returns recruiter accounts.
func (s *AdminService) ListRecruiters(ctx context.Context, limit, offset int) ([]domain.User, error) {
	return s.users.ListByRole(ctx, domain.RoleRecruiter, limit, offset)
}

// SetUserActive enables or disables a user account.
func (s *AdminService) SetUserActive(ctx context.Context, id int64, active bool) error {
	return s.users.SetActive(ctx, id, active)
}

// Analytics returns high-level hiring metrics.
func (s *AdminService) Analytics(ctx context.Context) (map[string]any, error) {
	jobsByStatus, err := s.jobs.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}
	appsByStatus, err := s.apps.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"jobs_by_status":         jobsByStatus,
		"applications_by_status": appsByStatus,
	}, nil
}
