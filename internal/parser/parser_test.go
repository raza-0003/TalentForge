package parser

import "testing"

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func TestExtractFields(t *testing.T) {
	text := "Contact: jane.doe@example.com or +1 (555) 123-4567.\n" +
		"Skills: Go, PostgreSQL, Docker and some React."
	f := ExtractFields(text)

	if !contains(f.Emails, "jane.doe@example.com") {
		t.Errorf("emails = %v, want jane.doe@example.com", f.Emails)
	}
	if len(f.Phones) == 0 {
		t.Error("expected at least one phone")
	}
	for _, want := range []string{"Go", "PostgreSQL", "Docker", "React"} {
		if !contains(f.Skills, want) {
			t.Errorf("skills = %v, missing %q", f.Skills, want)
		}
	}
}

func TestSkillMatchingAvoidsFalsePositives(t *testing.T) {
	// Case-sensitive matching: lowercase "go" and "goal" must not match "Go";
	// "Java" must not match inside "JavaScript".
	f := ExtractFields("I want to go to my goal using JavaScript.")
	if contains(f.Skills, "Go") {
		t.Errorf("unexpected Go match in %v", f.Skills)
	}
	if contains(f.Skills, "Java") {
		t.Errorf("unexpected Java match in %v", f.Skills)
	}
	if !contains(f.Skills, "JavaScript") {
		t.Errorf("expected JavaScript in %v", f.Skills)
	}
}

func TestCanonicalize(t *testing.T) {
	got := Canonicalize([]string{"go", "GOLANG", "postgresql", "CustomThing", "GO"})
	want := []string{"Go", "Golang", "PostgreSQL", "CustomThing"}
	if len(got) != len(want) {
		t.Fatalf("Canonicalize len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Canonicalize[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExtractTextPlain(t *testing.T) {
	text, method, err := ExtractText("text/plain", "resume.txt", []byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	if method != "text" || text != "hello world" {
		t.Errorf("got (%q, %q), want (hello world, text)", text, method)
	}
}

func TestExtractTextUnknown(t *testing.T) {
	_, method, err := ExtractText("application/octet-stream", "resume.bin", []byte{0x00, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	if method != "none" {
		t.Errorf("method = %q, want none", method)
	}
}
