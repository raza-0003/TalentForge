// Package service holds the application's business logic.
package service

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Notification is a message to be delivered out-of-band (email, etc.).
type Notification struct {
	Kind    string
	To      string
	Subject string
	Body    string
}

// Notifier dispatches notifications. It is backed by the Asynq enqueuer (which
// persists an outbox row and queues delivery); LogNotifier is a dev fallback.
type Notifier interface {
	Notify(ctx context.Context, n Notification)
}

// ReminderScheduler schedules a delayed reminder for an interview. The Asynq
// enqueuer implements it by queuing a task to run at remindAt.
type ReminderScheduler interface {
	ScheduleInterviewReminder(ctx context.Context, interviewID int64, remindAt time.Time)
}

// LogNotifier logs notifications instead of sending them.
type LogNotifier struct{ log *zap.Logger }

// NewLogNotifier builds a LogNotifier.
func NewLogNotifier(log *zap.Logger) *LogNotifier { return &LogNotifier{log: log} }

// Notify records the notification.
func (l *LogNotifier) Notify(_ context.Context, n Notification) {
	l.log.Info("notification queued",
		zap.String("kind", n.Kind),
		zap.String("to", n.To),
		zap.String("subject", n.Subject),
	)
}
