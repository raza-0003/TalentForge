package domain

import "testing"

func TestRoleValid(t *testing.T) {
	for _, r := range []Role{RoleCandidate, RoleRecruiter, RoleAdmin} {
		if !r.Valid() {
			t.Errorf("%q should be valid", r)
		}
	}
	if Role("superuser").Valid() {
		t.Error("unknown role should be invalid")
	}
}

func TestApplicationStatusValid(t *testing.T) {
	for _, s := range []ApplicationStatus{AppApplied, AppShortlisted, AppHired, AppRejected} {
		if !s.Valid() {
			t.Errorf("%q should be valid", s)
		}
	}
	if ApplicationStatus("bogus").Valid() {
		t.Error("unknown status should be invalid")
	}
}

func TestInterviewAndRecommendationValid(t *testing.T) {
	if !ModeVideo.Valid() || !IntScheduled.Valid() || !RecStrongYes.Valid() {
		t.Error("known enum values should be valid")
	}
	if InterviewMode("hologram").Valid() {
		t.Error("unknown mode should be invalid")
	}
	if Recommendation("maybe").Valid() {
		t.Error("unknown recommendation should be invalid")
	}
}
