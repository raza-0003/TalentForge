package domain

import "time"

// JobStatus is the lifecycle state of a job posting.
type JobStatus string

const (
	JobDraft  JobStatus = "draft"
	JobOpen   JobStatus = "open"
	JobClosed JobStatus = "closed"
)

// Job is a role a recruiter is hiring for.
type Job struct {
	ID             int64     `json:"id"`
	CreatedBy      int64     `json:"created_by"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Department     string    `json:"department,omitempty"`
	Location       string    `json:"location,omitempty"`
	EmploymentType string    `json:"employment_type,omitempty"`
	MinExperience  int       `json:"min_experience"`
	Status         JobStatus `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ApplicationStatus is the stage a candidate has reached for a job.
type ApplicationStatus string

const (
	AppApplied     ApplicationStatus = "applied"
	AppScreening   ApplicationStatus = "screening"
	AppShortlisted ApplicationStatus = "shortlisted"
	AppInterview   ApplicationStatus = "interview"
	AppOffer       ApplicationStatus = "offer"
	AppHired       ApplicationStatus = "hired"
	AppRejected    ApplicationStatus = "rejected"
	AppWithdrawn   ApplicationStatus = "withdrawn"
)

// Valid reports whether s is a known application status.
func (s ApplicationStatus) Valid() bool {
	switch s {
	case AppApplied, AppScreening, AppShortlisted, AppInterview,
		AppOffer, AppHired, AppRejected, AppWithdrawn:
		return true
	}
	return false
}

// Application is a candidate's application to a job.
type Application struct {
	ID          int64             `json:"id"`
	JobID       int64             `json:"job_id"`
	CandidateID int64             `json:"candidate_id"`
	ResumeID    *int64            `json:"resume_id,omitempty"`
	Status      ApplicationStatus `json:"status"`
	CoverLetter string            `json:"cover_letter,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`

	// Optional joined fields, populated by list queries.
	JobTitle      string `json:"job_title,omitempty"`
	CandidateName string `json:"candidate_name,omitempty"`
}

// ApplicationEvent is one entry in an application's append-only timeline.
type ApplicationEvent struct {
	ID            int64              `json:"id"`
	ApplicationID int64              `json:"application_id"`
	ActorID       *int64             `json:"actor_id,omitempty"`
	EventType     string             `json:"event_type"`
	FromStatus    *ApplicationStatus `json:"from_status,omitempty"`
	ToStatus      *ApplicationStatus `json:"to_status,omitempty"`
	Note          string             `json:"note,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

// InterviewMode is how an interview is conducted.
type InterviewMode string

const (
	ModeOnsite InterviewMode = "onsite"
	ModePhone  InterviewMode = "phone"
	ModeVideo  InterviewMode = "video"
)

// Valid reports whether m is a known interview mode.
func (m InterviewMode) Valid() bool {
	switch m {
	case ModeOnsite, ModePhone, ModeVideo:
		return true
	}
	return false
}

// InterviewStatus is the lifecycle state of an interview.
type InterviewStatus string

const (
	IntScheduled InterviewStatus = "scheduled"
	IntCompleted InterviewStatus = "completed"
	IntCancelled InterviewStatus = "cancelled"
	IntNoShow    InterviewStatus = "no_show"
)

// Valid reports whether s is a known interview status.
func (s InterviewStatus) Valid() bool {
	switch s {
	case IntScheduled, IntCompleted, IntCancelled, IntNoShow:
		return true
	}
	return false
}

// Interview is a scheduled conversation tied to an application.
type Interview struct {
	ID              int64           `json:"id"`
	ApplicationID   int64           `json:"application_id"`
	InterviewerID   int64           `json:"interviewer_id"`
	CreatedBy       int64           `json:"created_by"`
	ScheduledAt     time.Time       `json:"scheduled_at"`
	DurationMinutes int             `json:"duration_minutes"`
	Mode            InterviewMode   `json:"mode"`
	Location        string          `json:"location,omitempty"`
	Status          InterviewStatus `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Recommendation is an interviewer's hiring recommendation.
type Recommendation string

const (
	RecStrongYes Recommendation = "strong_yes"
	RecYes       Recommendation = "yes"
	RecNo        Recommendation = "no"
	RecStrongNo  Recommendation = "strong_no"
)

// Valid reports whether r is a known recommendation.
func (r Recommendation) Valid() bool {
	switch r {
	case RecStrongYes, RecYes, RecNo, RecStrongNo:
		return true
	}
	return false
}

// OfferStatus is the lifecycle state of an offer letter.
type OfferStatus string

const (
	OfferDraft     OfferStatus = "draft"
	OfferSent      OfferStatus = "sent"
	OfferAccepted  OfferStatus = "accepted"
	OfferDeclined  OfferStatus = "declined"
	OfferRescinded OfferStatus = "rescinded"
)

// Offer is an offer letter tied to an application.
type Offer struct {
	ID             int64       `json:"id"`
	ApplicationID  int64       `json:"application_id"`
	CreatedBy      int64       `json:"created_by"`
	PositionTitle  string      `json:"position_title"`
	SalaryAmount   *float64    `json:"salary_amount,omitempty"`
	SalaryCurrency string      `json:"salary_currency"`
	StartDate      *time.Time  `json:"start_date,omitempty"`
	Status         OfferStatus `json:"status"`
	StorageKey     string      `json:"storage_key,omitempty"`
	ExpiresAt      *time.Time  `json:"expires_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// Feedback is an interviewer's structured evaluation.
type Feedback struct {
	ID             int64          `json:"id"`
	InterviewID    int64          `json:"interview_id"`
	AuthorID       int64          `json:"author_id"`
	Rating         int            `json:"rating"`
	Recommendation Recommendation `json:"recommendation"`
	Strengths      string         `json:"strengths,omitempty"`
	Weaknesses     string         `json:"weaknesses,omitempty"`
	Comments       string         `json:"comments,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
