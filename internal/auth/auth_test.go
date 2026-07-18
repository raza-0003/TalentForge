package auth

import (
	"testing"
	"time"

	"github.com/faizan/ats/internal/domain"
)

func TestPasswordHashing(t *testing.T) {
	h, err := HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if h == "password123" {
		t.Error("hash must not equal the plaintext")
	}
	if !CheckPassword(h, "password123") {
		t.Error("correct password was rejected")
	}
	if CheckPassword(h, "wrong-password") {
		t.Error("incorrect password was accepted")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	tm := NewTokenManager("test-secret", time.Minute, time.Hour)
	u := &domain.User{ID: 42, Role: domain.RoleRecruiter}

	tok, err := tm.GenerateAccessToken(u)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tm.ParseAccessToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Role != domain.RoleRecruiter {
		t.Errorf("Role = %q, want recruiter", claims.Role)
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	tm := NewTokenManager("test-secret", -time.Minute, time.Hour) // already expired
	tok, err := tm.GenerateAccessToken(&domain.User{ID: 1, Role: domain.RoleCandidate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tm.ParseAccessToken(tok); err == nil {
		t.Error("expected an error for an expired token")
	}
}

func TestWrongSecretRejected(t *testing.T) {
	signer := NewTokenManager("secret-a", time.Minute, time.Hour)
	verifier := NewTokenManager("secret-b", time.Minute, time.Hour)
	tok, _ := signer.GenerateAccessToken(&domain.User{ID: 1, Role: domain.RoleAdmin})
	if _, err := verifier.ParseAccessToken(tok); err == nil {
		t.Error("expected a signature-validation error")
	}
}

func TestRefreshTokenHashing(t *testing.T) {
	raw, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || hash == "" {
		t.Fatal("empty token or hash")
	}
	if raw == hash {
		t.Error("raw token and stored hash must differ")
	}
	if HashToken(raw) != hash {
		t.Error("HashToken must reproduce the stored hash")
	}
}
