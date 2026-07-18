package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/faizan/ats/internal/auth"
	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/repository"
)

// AuthService handles registration, login, and token lifecycle.
type AuthService struct {
	users  *repository.UserRepo
	tokens *repository.RefreshTokenRepo
	tm     *auth.TokenManager
}

// NewAuthService builds an AuthService.
func NewAuthService(users *repository.UserRepo, tokens *repository.RefreshTokenRepo, tm *auth.TokenManager) *AuthService {
	return &AuthService{users: users, tokens: tokens, tm: tm}
}

// TokenPair is returned to clients on login/refresh.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Signup registers a new candidate. Recruiters and admins are provisioned by an
// admin, never via self-signup.
func (s *AuthService) Signup(ctx context.Context, email, password, fullName string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(password) < 8 || strings.TrimSpace(fullName) == "" {
		return nil, fmt.Errorf("%w: email, name, and an 8+ character password are required", domain.ErrValidation)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &domain.User{Email: email, FullName: fullName, Role: domain.RoleCandidate, PasswordHash: hash}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Login verifies credentials and issues a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, *TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !u.IsActive || !auth.CheckPassword(u.PasswordHash, password) {
		return nil, nil, domain.ErrInvalidCredentials
	}
	pair, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	return u, pair, nil
}

// Refresh rotates a refresh token: the old one is consumed and a new pair issued.
func (s *AuthService) Refresh(ctx context.Context, rawRefresh string) (*TokenPair, error) {
	userID, err := s.tokens.Consume(ctx, auth.HashToken(rawRefresh))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, u)
}

// Logout revokes a refresh token.
func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	return s.tokens.Revoke(ctx, auth.HashToken(rawRefresh))
}

// Me returns the current user.
func (s *AuthService) Me(ctx context.Context, id int64) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *AuthService) issueTokens(ctx context.Context, u *domain.User) (*TokenPair, error) {
	access, err := s.tm.GenerateAccessToken(u)
	if err != nil {
		return nil, err
	}
	raw, hash, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := s.tokens.Store(ctx, u.ID, hash, time.Now().Add(s.tm.RefreshTTL())); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: raw,
		ExpiresAt:    time.Now().Add(s.tm.AccessTTL()),
	}, nil
}
