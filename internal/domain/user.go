package domain

import (
	"encoding/json"
	"time"
)

// Role is a user's access role.
type Role string

const (
	RoleCandidate Role = "candidate"
	RoleRecruiter Role = "recruiter"
	RoleAdmin     Role = "admin"
)

// Valid reports whether r is a known role.
func (r Role) Valid() bool {
	switch r {
	case RoleCandidate, RoleRecruiter, RoleAdmin:
		return true
	}
	return false
}

// User is an account. PasswordHash is never serialized to JSON.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	Role         Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CandidateProfile is the extended profile for a candidate user.
type CandidateProfile struct {
	ID        int64             `json:"id"`
	UserID    int64             `json:"user_id"`
	Phone     string            `json:"phone,omitempty"`
	Headline  string            `json:"headline,omitempty"`
	Location  string            `json:"location,omitempty"`
	Links     map[string]string `json:"links"`
	Skills    []string          `json:"skills"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// CandidateSearchResult is a row returned by recruiter candidate search.
type CandidateSearchResult struct {
	UserID   int64    `json:"user_id"`
	FullName string   `json:"full_name"`
	Email    string   `json:"email"`
	Headline string   `json:"headline,omitempty"`
	Location string   `json:"location,omitempty"`
	Skills   []string `json:"skills"`
}

// Resume is metadata about an uploaded resume file.
type Resume struct {
	ID              int64      `json:"id"`
	CandidateUserID int64      `json:"candidate_user_id"`
	StorageKey      string     `json:"storage_key"`
	FileName        string     `json:"file_name"`
	ContentType     string     `json:"content_type,omitempty"`
	SizeBytes       int64      `json:"size_bytes,omitempty"`
	IsPrimary       bool            `json:"is_primary"`
	ParsedAt        *time.Time      `json:"parsed_at,omitempty"`
	ParsedData      json.RawMessage `json:"parsed_data,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}
