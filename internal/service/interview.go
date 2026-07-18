package service

import (
	"context"
	"fmt"
	"time"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/repository"
)

// InterviewService handles scheduling interviews and recording feedback.
type InterviewService struct {
	interviews *repository.InterviewRepo
	feedback   *repository.FeedbackRepo
	apps       *repository.ApplicationRepo
	users      *repository.UserRepo
	notif      Notifier
	reminders  ReminderScheduler
}

// NewInterviewService builds an InterviewService.
func NewInterviewService(interviews *repository.InterviewRepo, feedback *repository.FeedbackRepo,
	apps *repository.ApplicationRepo, users *repository.UserRepo,
	notif Notifier, reminders ReminderScheduler) *InterviewService {
	return &InterviewService{
		interviews: interviews, feedback: feedback, apps: apps,
		users: users, notif: notif, reminders: reminders,
	}
}

// Schedule books an interview for an application and records a timeline event.
func (s *InterviewService) Schedule(ctx context.Context, createdBy, appID, interviewerID int64,
	scheduledAt time.Time, duration int, mode domain.InterviewMode, location string) (*domain.Interview, error) {
	if _, err := s.apps.GetByID(ctx, appID); err != nil {
		return nil, err
	}
	if scheduledAt.Before(time.Now()) {
		return nil, fmt.Errorf("%w: scheduled_at must be in the future", domain.ErrValidation)
	}
	if mode == "" {
		mode = domain.ModeVideo
	}
	if !mode.Valid() {
		return nil, fmt.Errorf("%w: invalid interview mode", domain.ErrValidation)
	}
	if duration <= 0 {
		duration = 60
	}

	iv := &domain.Interview{
		ApplicationID:   appID,
		InterviewerID:   interviewerID,
		CreatedBy:       createdBy,
		ScheduledAt:     scheduledAt,
		DurationMinutes: duration,
		Mode:            mode,
		Location:        location,
		Status:          domain.IntScheduled,
	}
	if err := s.interviews.Create(ctx, iv); err != nil {
		return nil, err
	}

	_ = s.apps.AddEvent(ctx, &domain.ApplicationEvent{
		ApplicationID: appID,
		ActorID:       &createdBy,
		EventType:     "interview_scheduled",
		Note:          "Interview on " + scheduledAt.Format(time.RFC1123),
	})

	if u, err := s.users.GetByID(ctx, interviewerID); err == nil {
		s.notif.Notify(ctx, Notification{
			Kind:    "interview_scheduled",
			To:      u.Email,
			Subject: "Interview scheduled",
			Body:    "You are scheduled to interview a candidate on " + scheduledAt.Format(time.RFC1123) + ".",
		})
	}

	// Schedule a reminder 24h before the interview (or shortly from now if the
	// interview is sooner than that).
	if s.reminders != nil {
		remindAt := scheduledAt.Add(-24 * time.Hour)
		if !remindAt.After(time.Now()) {
			remindAt = time.Now().Add(1 * time.Minute)
		}
		s.reminders.ScheduleInterviewReminder(ctx, iv.ID, remindAt)
	}
	return iv, nil
}

// ListByApplication returns all interviews for an application.
func (s *InterviewService) ListByApplication(ctx context.Context, appID int64) ([]domain.Interview, error) {
	return s.interviews.ListByApplication(ctx, appID)
}

// Update reschedules or changes the status of an interview. Zero-valued fields
// are left unchanged.
func (s *InterviewService) Update(ctx context.Context, id int64, scheduledAt time.Time, duration int,
	mode domain.InterviewMode, location string, status domain.InterviewStatus) (*domain.Interview, error) {
	iv, err := s.interviews.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !scheduledAt.IsZero() {
		iv.ScheduledAt = scheduledAt
	}
	if duration > 0 {
		iv.DurationMinutes = duration
	}
	if mode != "" {
		if !mode.Valid() {
			return nil, fmt.Errorf("%w: invalid interview mode", domain.ErrValidation)
		}
		iv.Mode = mode
	}
	if location != "" {
		iv.Location = location
	}
	if status != "" {
		if !status.Valid() {
			return nil, fmt.Errorf("%w: invalid interview status", domain.ErrValidation)
		}
		iv.Status = status
	}
	if err := s.interviews.Update(ctx, iv.ID, iv.ScheduledAt, iv.DurationMinutes, iv.Mode, iv.Location, iv.Status); err != nil {
		return nil, err
	}
	return iv, nil
}

// AddFeedback records an interviewer's evaluation. Returns ErrConflict if
// feedback already exists for the interview.
func (s *InterviewService) AddFeedback(ctx context.Context, authorID, interviewID int64, rating int,
	rec domain.Recommendation, strengths, weaknesses, comments string) (*domain.Feedback, error) {
	if _, err := s.interviews.GetByID(ctx, interviewID); err != nil {
		return nil, err
	}
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("%w: rating must be between 1 and 5", domain.ErrValidation)
	}
	if !rec.Valid() {
		return nil, fmt.Errorf("%w: invalid recommendation", domain.ErrValidation)
	}
	f := &domain.Feedback{
		InterviewID:    interviewID,
		AuthorID:       authorID,
		Rating:         rating,
		Recommendation: rec,
		Strengths:      strengths,
		Weaknesses:     weaknesses,
		Comments:       comments,
	}
	if err := s.feedback.Create(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// GetFeedback returns the feedback for an interview.
func (s *InterviewService) GetFeedback(ctx context.Context, interviewID int64) (*domain.Feedback, error) {
	return s.feedback.GetByInterview(ctx, interviewID)
}
