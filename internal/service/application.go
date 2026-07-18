package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/repository"
)

// ApplicationService handles applying, status changes, and the timeline.
type ApplicationService struct {
	apps    *repository.ApplicationRepo
	jobs    *repository.JobRepo
	users   *repository.UserRepo
	resumes *repository.ResumeRepo
	notif   Notifier
}

// NewApplicationService builds an ApplicationService.
func NewApplicationService(apps *repository.ApplicationRepo, jobs *repository.JobRepo,
	users *repository.UserRepo, resumes *repository.ResumeRepo, notif Notifier) *ApplicationService {
	return &ApplicationService{apps: apps, jobs: jobs, users: users, resumes: resumes, notif: notif}
}

func ptrStatus(s domain.ApplicationStatus) *domain.ApplicationStatus { return &s }

// Apply creates an application for an open job, optionally attaching one of the
// candidate's own resumes. Returns ErrConflict if the candidate already applied.
func (s *ApplicationService) Apply(ctx context.Context, candidateID, jobID int64, resumeID *int64, coverLetter string) (*domain.Application, error) {
	job, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != domain.JobOpen {
		return nil, fmt.Errorf("%w: this job is not open for applications", domain.ErrValidation)
	}

	if resumeID != nil {
		res, err := s.resumes.GetByID(ctx, *resumeID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("%w: resume not found", domain.ErrValidation)
			}
			return nil, err
		}
		if res.CandidateUserID != candidateID {
			return nil, domain.ErrForbidden
		}
	}

	app := &domain.Application{
		JobID:       jobID,
		CandidateID: candidateID,
		ResumeID:    resumeID,
		Status:      domain.AppApplied,
		CoverLetter: coverLetter,
	}
	if err := s.apps.Create(ctx, app); err != nil {
		return nil, err
	}

	_ = s.apps.AddEvent(ctx, &domain.ApplicationEvent{
		ApplicationID: app.ID,
		ActorID:       &candidateID,
		EventType:     "created",
		ToStatus:      ptrStatus(domain.AppApplied),
	})

	if u, err := s.users.GetByID(ctx, candidateID); err == nil {
		s.notif.Notify(ctx, Notification{
			Kind:    "application_received",
			To:      u.Email,
			Subject: "We received your application: " + job.Title,
			Body:    "Hi " + u.FullName + ", thanks for applying to " + job.Title + ". We'll be in touch.",
		})
	}
	return app, nil
}

// Get returns a single application.
func (s *ApplicationService) Get(ctx context.Context, id int64) (*domain.Application, error) {
	return s.apps.GetByID(ctx, id)
}

// ListByJob returns applications for a job (recruiter view).
func (s *ApplicationService) ListByJob(ctx context.Context, jobID int64, status domain.ApplicationStatus, limit, offset int) ([]domain.Application, error) {
	return s.apps.ListByJob(ctx, jobID, status, limit, offset)
}

// ListByCandidate returns a candidate's own applications.
func (s *ApplicationService) ListByCandidate(ctx context.Context, candidateID int64, limit, offset int) ([]domain.Application, error) {
	return s.apps.ListByCandidate(ctx, candidateID, limit, offset)
}

// Timeline returns an application's event history.
func (s *ApplicationService) Timeline(ctx context.Context, appID int64) ([]domain.ApplicationEvent, error) {
	return s.apps.ListEvents(ctx, appID)
}

// ChangeStatus advances an application, records a timeline event, and notifies
// the candidate.
func (s *ApplicationService) ChangeStatus(ctx context.Context, actorID, appID int64, to domain.ApplicationStatus, note string) (*domain.Application, error) {
	if !to.Valid() {
		return nil, fmt.Errorf("%w: unknown status %q", domain.ErrValidation, to)
	}
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app.Status == to {
		return app, nil
	}
	from := app.Status
	if err := s.apps.UpdateStatus(ctx, appID, to); err != nil {
		return nil, err
	}
	_ = s.apps.AddEvent(ctx, &domain.ApplicationEvent{
		ApplicationID: appID,
		ActorID:       &actorID,
		EventType:     "status_changed",
		FromStatus:    &from,
		ToStatus:      &to,
		Note:          note,
	})
	app.Status = to

	if u, err := s.users.GetByID(ctx, app.CandidateID); err == nil {
		s.notif.Notify(ctx, Notification{
			Kind:    "status_changed",
			To:      u.Email,
			Subject: "Update on your application",
			Body:    "Your application status is now: " + string(to),
		})
	}
	return app, nil
}
