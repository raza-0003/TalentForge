package service

import (
	"context"
	"errors"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/parser"
	"github.com/faizan/ats/internal/repository"
)

// CandidateService handles candidate profile reads and writes.
type CandidateService struct {
	profiles *repository.ProfileRepo
}

// NewCandidateService builds a CandidateService.
func NewCandidateService(profiles *repository.ProfileRepo) *CandidateService {
	return &CandidateService{profiles: profiles}
}

// GetProfile returns the candidate's profile, or an empty editable shell if
// they haven't created one yet.
func (s *CandidateService) GetProfile(ctx context.Context, userID int64) (*domain.CandidateProfile, error) {
	p, err := s.profiles.Get(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return &domain.CandidateProfile{
			UserID: userID,
			Links:  map[string]string{},
			Skills: []string{},
		}, nil
	}
	return p, err
}

// UpdateProfile creates or updates the candidate's profile. Skills are
// canonicalized so search matches regardless of the casing the candidate typed.
func (s *CandidateService) UpdateProfile(ctx context.Context, p *domain.CandidateProfile) error {
	if p.Links == nil {
		p.Links = map[string]string{}
	}
	p.Skills = parser.Canonicalize(p.Skills)
	return s.profiles.Upsert(ctx, p)
}

// Search finds candidates by skills and/or a name/headline query (recruiter
// feature). Search terms are canonicalized to match stored skills.
func (s *CandidateService) Search(ctx context.Context, skills []string, query string, matchAll bool, limit, offset int) ([]domain.CandidateSearchResult, error) {
	return s.profiles.Search(ctx, parser.Canonicalize(skills), query, matchAll, limit, offset)
}
