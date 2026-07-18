package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/mailer"
	"github.com/faizan/ats/internal/parser"
	"github.com/faizan/ats/internal/repository"
	"github.com/faizan/ats/internal/storage"
)

// sweepBatchLimit caps how many stuck notifications one sweep re-attempts.
const sweepBatchLimit = 200

// Handlers is the queue consumer: it processes tasks in the worker process.
type Handlers struct {
	mailer     mailer.Mailer
	notifs     *repository.NotificationRepo
	interviews *repository.InterviewRepo
	users      *repository.UserRepo
	resumes    *repository.ResumeRepo
	store      storage.Storage
	sweepAfter time.Duration
	log        *zap.Logger
}

// NewHandlers builds the worker task handlers.
func NewHandlers(m mailer.Mailer, notifs *repository.NotificationRepo,
	interviews *repository.InterviewRepo, users *repository.UserRepo,
	resumes *repository.ResumeRepo, store storage.Storage,
	sweepAfter time.Duration, log *zap.Logger) *Handlers {
	return &Handlers{
		mailer: m, notifs: notifs, interviews: interviews, users: users,
		resumes: resumes, store: store, sweepAfter: sweepAfter, log: log,
	}
}

// Register mounts every task handler on the Asynq mux.
func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEmailSend, h.HandleEmail)
	mux.HandleFunc(TypeInterviewReminder, h.HandleInterviewReminder)
	mux.HandleFunc(TypeSweepPending, h.HandleSweepPending)
	mux.HandleFunc(TypeResumeParse, h.HandleResumeParse)
}

// HandleEmail delivers a queued email and updates the outbox row.
func (h *Handlers) HandleEmail(ctx context.Context, t *asynq.Task) error {
	var p EmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal email payload: %v: %w", err, asynq.SkipRetry)
	}

	if err := h.mailer.Send(ctx, mailer.Message{To: p.To, Subject: p.Subject, Body: p.Body}); err != nil {
		if p.NotificationID != 0 {
			_ = h.notifs.MarkFailed(ctx, p.NotificationID, err.Error())
		}
		return fmt.Errorf("send email: %w", err)
	}

	if p.NotificationID != 0 {
		_ = h.notifs.MarkSent(ctx, p.NotificationID)
	}
	h.log.Info("email sent", zap.String("to", p.To), zap.String("kind", p.Kind))
	return nil
}

// HandleInterviewReminder emails the interviewer ahead of a scheduled interview.
func (h *Handlers) HandleInterviewReminder(ctx context.Context, t *asynq.Task) error {
	var p InterviewReminderPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal reminder payload: %v: %w", err, asynq.SkipRetry)
	}

	iv, err := h.interviews.GetByID(ctx, p.InterviewID)
	if err != nil {
		return fmt.Errorf("load interview %d: %v: %w", p.InterviewID, err, asynq.SkipRetry)
	}
	if iv.Status == domain.IntCancelled {
		h.log.Info("skipping reminder for cancelled interview", zap.Int64("interview_id", iv.ID))
		return nil
	}

	u, err := h.users.GetByID(ctx, iv.InterviewerID)
	if err != nil {
		return fmt.Errorf("load interviewer: %w", err)
	}
	return h.mailer.Send(ctx, mailer.Message{
		To:      u.Email,
		Subject: "Reminder: upcoming interview",
		Body: fmt.Sprintf("Hi %s, this is a reminder that you have an interview scheduled for %s.",
			u.FullName, iv.ScheduledAt.Format(time.RFC1123)),
	})
}

// HandleSweepPending re-attempts delivery of notifications stuck in 'pending'.
func (h *Handlers) HandleSweepPending(ctx context.Context, _ *asynq.Task) error {
	olderThan := time.Now().Add(-h.sweepAfter)
	pending, err := h.notifs.ListPending(ctx, olderThan, sweepBatchLimit)
	if err != nil {
		return fmt.Errorf("list pending notifications: %w", err)
	}
	if len(pending) == 0 {
		return nil
	}

	var sent, failed int
	for _, n := range pending {
		if err := h.mailer.Send(ctx, mailer.Message{To: n.RecipientEmail, Subject: n.Subject, Body: n.Body}); err != nil {
			_ = h.notifs.MarkFailed(ctx, n.ID, "sweep: "+err.Error())
			failed++
			continue
		}
		_ = h.notifs.MarkSent(ctx, n.ID)
		sent++
	}
	h.log.Info("swept pending notifications",
		zap.Int("candidates", len(pending)), zap.Int("sent", sent), zap.Int("failed", failed))
	return nil
}

// HandleResumeParse reads a stored resume, extracts text and fields, and saves
// the result to resumes.parsed_data.
func (h *Handlers) HandleResumeParse(ctx context.Context, t *asynq.Task) error {
	var p ResumeParsePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal resume parse payload: %v: %w", err, asynq.SkipRetry)
	}

	res, err := h.resumes.GetByID(ctx, p.ResumeID)
	if err != nil {
		return fmt.Errorf("load resume %d: %v: %w", p.ResumeID, err, asynq.SkipRetry)
	}

	rc, err := h.store.Open(ctx, res.StorageKey)
	if err != nil {
		return fmt.Errorf("open resume file: %w", err)
	}
	data, readErr := io.ReadAll(rc)
	_ = rc.Close()
	if readErr != nil {
		return fmt.Errorf("read resume file: %w", readErr)
	}

	parsed, err := parser.Parse(res.ContentType, res.FileName, data)
	if err != nil {
		return fmt.Errorf("parse resume: %w", err)
	}
	b, err := json.Marshal(parsed)
	if err != nil {
		return fmt.Errorf("marshal parsed resume: %v: %w", err, asynq.SkipRetry)
	}
	if err := h.resumes.SetParsed(ctx, res.ID, b); err != nil {
		return fmt.Errorf("save parsed resume: %w", err)
	}

	h.log.Info("resume parsed",
		zap.Int64("resume_id", res.ID),
		zap.String("extractor", parsed.Extractor),
		zap.Int("skills", len(parsed.Skills)),
	)
	return nil
}
