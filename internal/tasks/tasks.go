// Package tasks defines the background task types plus the enqueuer (producer)
// and handlers (consumer) for the Asynq/Redis queue.
package tasks

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Task type names.
const (
	TypeEmailSend         = "email:send"
	TypeInterviewReminder = "interview:reminder"
	TypeSweepPending      = "notifications:sweep"
	TypeResumeParse       = "resume:parse"
)

// EmailPayload is the payload for an email:send task.
type EmailPayload struct {
	NotificationID int64  `json:"notification_id,omitempty"` // outbox row to mark sent/failed
	To             string `json:"to"`
	Subject        string `json:"subject"`
	Body           string `json:"body"`
	Kind           string `json:"kind"`
}

// NewEmailTask builds an email:send task.
func NewEmailTask(p EmailPayload) (*asynq.Task, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal email payload: %w", err)
	}
	return asynq.NewTask(TypeEmailSend, b, asynq.MaxRetry(5)), nil
}

// InterviewReminderPayload is the payload for an interview:reminder task.
type InterviewReminderPayload struct {
	InterviewID int64 `json:"interview_id"`
}

// NewInterviewReminderTask builds a reminder task and the options to run it at
// processAt.
func NewInterviewReminderTask(interviewID int64, processAt time.Time) (*asynq.Task, []asynq.Option, error) {
	b, err := json.Marshal(InterviewReminderPayload{InterviewID: interviewID})
	if err != nil {
		return nil, nil, fmt.Errorf("marshal reminder payload: %w", err)
	}
	opts := []asynq.Option{asynq.ProcessAt(processAt), asynq.MaxRetry(3)}
	return asynq.NewTask(TypeInterviewReminder, b), opts, nil
}

// NewSweepPendingTask builds the periodic sweep task (no payload). The Scheduler
// enqueues it on a cron interval.
func NewSweepPendingTask() *asynq.Task {
	return asynq.NewTask(TypeSweepPending, nil, asynq.MaxRetry(1))
}

// ResumeParsePayload is the payload for a resume:parse task.
type ResumeParsePayload struct {
	ResumeID int64 `json:"resume_id"`
}

// NewResumeParseTask builds a resume:parse task.
func NewResumeParseTask(resumeID int64) (*asynq.Task, error) {
	b, err := json.Marshal(ResumeParsePayload{ResumeID: resumeID})
	if err != nil {
		return nil, fmt.Errorf("marshal resume parse payload: %w", err)
	}
	return asynq.NewTask(TypeResumeParse, b, asynq.MaxRetry(3)), nil
}
