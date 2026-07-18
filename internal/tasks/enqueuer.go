package tasks

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/faizan/ats/internal/repository"
	"github.com/faizan/ats/internal/service"
)

// Enqueuer is the queue producer. It implements service.Notifier and
// service.ReminderScheduler, so the API enqueues work onto Redis for the worker
// to process.
type Enqueuer struct {
	client *asynq.Client
	notifs *repository.NotificationRepo
	log    *zap.Logger
}

// NewEnqueuer builds an Enqueuer.
func NewEnqueuer(client *asynq.Client, notifs *repository.NotificationRepo, log *zap.Logger) *Enqueuer {
	return &Enqueuer{client: client, notifs: notifs, log: log}
}

// Notify persists an outbox row and enqueues an email task. Failures are logged,
// never propagated — notifications must not block the core flow.
func (e *Enqueuer) Notify(ctx context.Context, n service.Notification) {
	id, err := e.notifs.Create(ctx, &repository.NewNotification{
		RecipientEmail: n.To,
		Type:           n.Kind,
		Subject:        n.Subject,
		Body:           n.Body,
	})
	if err != nil {
		e.log.Error("persist notification failed", zap.Error(err), zap.String("to", n.To))
		return
	}
	task, err := NewEmailTask(EmailPayload{
		NotificationID: id,
		To:             n.To,
		Subject:        n.Subject,
		Body:           n.Body,
		Kind:           n.Kind,
	})
	if err != nil {
		e.log.Error("build email task failed", zap.Error(err))
		return
	}
	if _, err := e.client.EnqueueContext(ctx, task); err != nil {
		e.log.Error("enqueue email failed", zap.Error(err), zap.String("to", n.To))
	}
}

// ScheduleInterviewReminder enqueues a reminder task to run at remindAt.
func (e *Enqueuer) ScheduleInterviewReminder(ctx context.Context, interviewID int64, remindAt time.Time) {
	task, opts, err := NewInterviewReminderTask(interviewID, remindAt)
	if err != nil {
		e.log.Error("build reminder task failed", zap.Error(err))
		return
	}
	if _, err := e.client.EnqueueContext(ctx, task, opts...); err != nil {
		e.log.Error("enqueue reminder failed", zap.Error(err), zap.Int64("interview_id", interviewID))
	}
}

// EnqueueResumeParse queues a resume for background parsing.
func (e *Enqueuer) EnqueueResumeParse(ctx context.Context, resumeID int64) {
	task, err := NewResumeParseTask(resumeID)
	if err != nil {
		e.log.Error("build resume parse task failed", zap.Error(err))
		return
	}
	if _, err := e.client.EnqueueContext(ctx, task); err != nil {
		e.log.Error("enqueue resume parse failed", zap.Error(err), zap.Int64("resume_id", resumeID))
	}
}

// Close releases the underlying Asynq client.
func (e *Enqueuer) Close() error { return e.client.Close() }
