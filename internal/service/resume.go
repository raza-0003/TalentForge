package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/repository"
	"github.com/faizan/ats/internal/storage"
)

const maxResumeBytes = 10 << 20 // 10 MiB

var allowedResumeExt = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".txt": true, ".md": true, ".rtf": true,
}

// ResumeParseEnqueuer queues a resume for background parsing.
type ResumeParseEnqueuer interface {
	EnqueueResumeParse(ctx context.Context, resumeID int64)
}

// ResumeService handles resume upload, listing, and retrieval.
type ResumeService struct {
	resumes *repository.ResumeRepo
	store   storage.Storage
	parser  ResumeParseEnqueuer
}

// NewResumeService builds a ResumeService.
func NewResumeService(resumes *repository.ResumeRepo, store storage.Storage, parser ResumeParseEnqueuer) *ResumeService {
	return &ResumeService{resumes: resumes, store: store, parser: parser}
}

// Upload validates and stores a resume file, records its metadata, and queues
// it for parsing.
func (s *ResumeService) Upload(ctx context.Context, userID int64, fileName, contentType string,
	size int64, r io.Reader, makePrimary bool) (*domain.Resume, error) {
	if size <= 0 {
		return nil, fmt.Errorf("%w: empty file", domain.ErrValidation)
	}
	if size > maxResumeBytes {
		return nil, fmt.Errorf("%w: file exceeds the 10 MB limit", domain.ErrValidation)
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	if !allowedResumeExt[ext] {
		return nil, fmt.Errorf("%w: unsupported file type %q (allowed: pdf, doc, docx, txt, md, rtf)", domain.ErrValidation, ext)
	}

	key, err := buildResumeKey(userID, fileName)
	if err != nil {
		return nil, err
	}
	if err := s.store.Save(ctx, key, r); err != nil {
		return nil, err
	}

	res := &domain.Resume{
		CandidateUserID: userID,
		StorageKey:      key,
		FileName:        fileName,
		ContentType:     contentType,
		SizeBytes:       size,
		IsPrimary:       makePrimary,
	}
	if err := s.resumes.Create(ctx, res); err != nil {
		return nil, err
	}
	if makePrimary {
		if err := s.resumes.SetPrimary(ctx, userID, res.ID); err != nil {
			return nil, err
		}
	}
	if s.parser != nil {
		s.parser.EnqueueResumeParse(ctx, res.ID)
	}
	return res, nil
}

// List returns a candidate's resumes.
func (s *ResumeService) List(ctx context.Context, userID int64) ([]domain.Resume, error) {
	return s.resumes.ListByCandidate(ctx, userID)
}

// Get returns a single resume by id.
func (s *ResumeService) Get(ctx context.Context, id int64) (*domain.Resume, error) {
	return s.resumes.GetByID(ctx, id)
}

// SetPrimary marks a resume as the candidate's primary.
func (s *ResumeService) SetPrimary(ctx context.Context, userID, resumeID int64) error {
	return s.resumes.SetPrimary(ctx, userID, resumeID)
}

// Open returns the stored file for download.
func (s *ResumeService) Open(ctx context.Context, res *domain.Resume) (io.ReadCloser, error) {
	return s.store.Open(ctx, res.StorageKey)
}

func buildResumeKey(userID int64, fileName string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate storage key: %w", err)
	}
	return fmt.Sprintf("resumes/%d/%s-%s", userID, hex.EncodeToString(buf), sanitizeName(fileName)), nil
}

// sanitizeName strips path components and unsafe characters from a filename.
func sanitizeName(name string) string {
	name = filepath.Base(name)
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" || out == "." || out == ".." {
		out = "resume"
	}
	return out
}
