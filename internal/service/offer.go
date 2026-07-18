package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/pdf"
	"github.com/faizan/ats/internal/repository"
	"github.com/faizan/ats/internal/storage"
)

// OfferService handles offer creation (with PDF generation) and lifecycle.
type OfferService struct {
	offers *repository.OfferRepo
	apps   *repository.ApplicationRepo
	users  *repository.UserRepo
	jobs   *repository.JobRepo
	store  storage.Storage
	notif  Notifier
}

// NewOfferService builds an OfferService.
func NewOfferService(offers *repository.OfferRepo, apps *repository.ApplicationRepo,
	users *repository.UserRepo, jobs *repository.JobRepo, store storage.Storage, notif Notifier) *OfferService {
	return &OfferService{offers: offers, apps: apps, users: users, jobs: jobs, store: store, notif: notif}
}

// OfferInput is the data for creating an offer.
type OfferInput struct {
	PositionTitle  string
	SalaryAmount   *float64
	SalaryCurrency string
	StartDate      *time.Time
	ExpiresAt      *time.Time
}

// Create records a draft offer, renders its PDF, and stores it.
func (s *OfferService) Create(ctx context.Context, createdBy, appID int64, in OfferInput) (*domain.Offer, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	candidate, err := s.users.GetByID(ctx, app.CandidateID)
	if err != nil {
		return nil, err
	}
	var jobTitle string
	if job, err := s.jobs.GetByID(ctx, app.JobID); err == nil {
		jobTitle = job.Title
	}

	title := strings.TrimSpace(in.PositionTitle)
	if title == "" {
		title = jobTitle
	}
	if title == "" {
		return nil, fmt.Errorf("%w: position_title is required", domain.ErrValidation)
	}
	currency := in.SalaryCurrency
	if currency == "" {
		currency = "USD"
	}

	o := &domain.Offer{
		ApplicationID:  appID,
		CreatedBy:      createdBy,
		PositionTitle:  title,
		SalaryAmount:   in.SalaryAmount,
		SalaryCurrency: currency,
		StartDate:      in.StartDate,
		ExpiresAt:      in.ExpiresAt,
		Status:         domain.OfferDraft,
	}
	if err := s.offers.Create(ctx, o); err != nil {
		return nil, err
	}

	// Render and store the PDF synchronously (pure CPU, no external calls).
	doc := pdf.Render(composeOfferLetter(o, candidate.FullName))
	key := fmt.Sprintf("offers/%d/offer-%d.pdf", appID, o.ID)
	if err := s.store.Save(ctx, key, bytes.NewReader(doc)); err != nil {
		return nil, err
	}
	if err := s.offers.SetStorageKey(ctx, o.ID, key); err != nil {
		return nil, err
	}
	o.StorageKey = key
	return o, nil
}

// Get returns a single offer.
func (s *OfferService) Get(ctx context.Context, id int64) (*domain.Offer, error) {
	return s.offers.GetByID(ctx, id)
}

// ListByApplication returns an application's offers (recruiter view).
func (s *OfferService) ListByApplication(ctx context.Context, appID int64) ([]domain.Offer, error) {
	return s.offers.ListByApplication(ctx, appID)
}

// ListByCandidate returns a candidate's offers.
func (s *OfferService) ListByCandidate(ctx context.Context, candidateID int64) ([]domain.Offer, error) {
	return s.offers.ListByCandidate(ctx, candidateID)
}

// Send delivers a draft offer to the candidate: moves the application to the
// offer stage, records a timeline event, and notifies the candidate.
func (s *OfferService) Send(ctx context.Context, actorID, offerID int64) (*domain.Offer, error) {
	o, err := s.offers.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	if o.Status != domain.OfferDraft {
		return nil, fmt.Errorf("%w: only draft offers can be sent", domain.ErrValidation)
	}
	if err := s.offers.SetStatus(ctx, offerID, domain.OfferSent); err != nil {
		return nil, err
	}
	o.Status = domain.OfferSent

	if app, err := s.apps.GetByID(ctx, o.ApplicationID); err == nil {
		if app.Status != domain.AppOffer {
			_ = s.apps.UpdateStatus(ctx, app.ID, domain.AppOffer)
		}
		_ = s.apps.AddEvent(ctx, &domain.ApplicationEvent{
			ApplicationID: app.ID, ActorID: &actorID, EventType: "offer_sent",
			ToStatus: ptrStatus(domain.AppOffer),
		})
		if u, err := s.users.GetByID(ctx, app.CandidateID); err == nil {
			s.notif.Notify(ctx, Notification{
				Kind:    "offer_sent",
				To:      u.Email,
				Subject: "You've received an offer: " + o.PositionTitle,
				Body:    "Congratulations! An offer is ready for you to review, download, and accept in the candidate portal.",
			})
		}
	}
	return o, nil
}

// Respond records a candidate's accept/decline on their own offer.
func (s *OfferService) Respond(ctx context.Context, candidateID, offerID int64, accept bool) (*domain.Offer, error) {
	o, err := s.offers.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	app, err := s.apps.GetByID(ctx, o.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app.CandidateID != candidateID {
		return nil, domain.ErrForbidden
	}
	if o.Status != domain.OfferSent {
		return nil, fmt.Errorf("%w: this offer is not awaiting a response", domain.ErrValidation)
	}

	newStatus, event := domain.OfferDeclined, "offer_declined"
	if accept {
		newStatus, event = domain.OfferAccepted, "offer_accepted"
	}
	if err := s.offers.SetStatus(ctx, offerID, newStatus); err != nil {
		return nil, err
	}
	o.Status = newStatus

	ev := &domain.ApplicationEvent{ApplicationID: app.ID, ActorID: &candidateID, EventType: event}
	if accept {
		_ = s.apps.UpdateStatus(ctx, app.ID, domain.AppHired)
		ev.ToStatus = ptrStatus(domain.AppHired)
	}
	_ = s.apps.AddEvent(ctx, ev)

	if u, err := s.users.GetByID(ctx, o.CreatedBy); err == nil {
		verb := "declined"
		if accept {
			verb = "accepted"
		}
		s.notif.Notify(ctx, Notification{
			Kind:    event,
			To:      u.Email,
			Subject: "Offer " + verb,
			Body:    "The candidate has " + verb + " the offer for " + o.PositionTitle + ".",
		})
	}
	return o, nil
}

// Rescind withdraws an offer that hasn't been accepted or declined.
func (s *OfferService) Rescind(ctx context.Context, actorID, offerID int64) (*domain.Offer, error) {
	o, err := s.offers.GetByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	if o.Status == domain.OfferAccepted || o.Status == domain.OfferDeclined {
		return nil, fmt.Errorf("%w: cannot rescind an offer already %s", domain.ErrValidation, o.Status)
	}
	if err := s.offers.SetStatus(ctx, offerID, domain.OfferRescinded); err != nil {
		return nil, err
	}
	o.Status = domain.OfferRescinded

	if app, err := s.apps.GetByID(ctx, o.ApplicationID); err == nil {
		_ = s.apps.AddEvent(ctx, &domain.ApplicationEvent{
			ApplicationID: app.ID, ActorID: &actorID, EventType: "offer_rescinded",
		})
		if u, err := s.users.GetByID(ctx, app.CandidateID); err == nil {
			s.notif.Notify(ctx, Notification{
				Kind: "offer_rescinded", To: u.Email,
				Subject: "Offer update", Body: "An offer previously extended to you has been rescinded.",
			})
		}
	}
	return o, nil
}

// OpenForDownload returns the offer PDF for the owning candidate or any
// recruiter/admin.
func (s *OfferService) OpenForDownload(ctx context.Context, offerID, userID int64, role domain.Role) (io.ReadCloser, *domain.Offer, error) {
	o, err := s.offers.GetByID(ctx, offerID)
	if err != nil {
		return nil, nil, err
	}
	if role == domain.RoleCandidate {
		app, err := s.apps.GetByID(ctx, o.ApplicationID)
		if err != nil {
			return nil, nil, err
		}
		if app.CandidateID != userID {
			return nil, nil, domain.ErrForbidden
		}
	}
	if o.StorageKey == "" {
		return nil, nil, domain.ErrNotFound
	}
	rc, err := s.store.Open(ctx, o.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return rc, o, nil
}

// composeOfferLetter builds the lines of the offer-letter PDF.
func composeOfferLetter(o *domain.Offer, candidateName string) []pdf.Line {
	var lines []pdf.Line
	add := func(text string, bold bool, size, gap float64) {
		lines = append(lines, pdf.Line{Text: text, Bold: bold, Size: size, Gap: gap})
	}
	para := func(text string, gapBefore float64) {
		for i, wl := range pdf.Wrap(text, 92) {
			gap := 0.0
			if i == 0 {
				gap = gapBefore
			}
			add(wl, false, 11, gap)
		}
	}

	add("Offer of Employment", true, 20, 0)
	add(o.CreatedAt.Format("January 2, 2006"), false, 10, 10)
	add("Dear "+candidateName+",", false, 11, 16)

	para(fmt.Sprintf("We are delighted to offer you the position of %s. We were impressed by "+
		"your background and are confident you will be a valuable addition to the team.", o.PositionTitle), 6)

	if o.SalaryAmount != nil {
		add(fmt.Sprintf("Annual compensation: %s %s", o.SalaryCurrency, formatMoney(*o.SalaryAmount)), false, 11, 12)
	}
	if o.StartDate != nil {
		add("Proposed start date: "+o.StartDate.Format("January 2, 2006"), false, 11, 4)
	}
	if o.ExpiresAt != nil {
		add("This offer is valid until: "+o.ExpiresAt.Format("January 2, 2006"), false, 11, 4)
	}

	para("This offer is contingent on standard background checks and the terms of your "+
		"employment agreement. Please review the offer and record your decision in the candidate portal.", 12)

	add("Sincerely,", false, 11, 20)
	add("The Hiring Team", false, 11, 4)
	return lines
}

// formatMoney renders a value with thousands separators, e.g. 120000 -> 120,000.00.
func formatMoney(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	dot := strings.IndexByte(s, '.')
	intPart, decPart := s[:dot], s[dot:]
	var b strings.Builder
	n := len(intPart)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(intPart[i])
	}
	out := b.String() + decPart
	if neg {
		out = "-" + out
	}
	return out
}
