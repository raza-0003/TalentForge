package pdf

import (
	"bytes"
	"testing"
)

func TestRenderStructure(t *testing.T) {
	out := Render([]Line{
		{Text: "Offer of Employment", Bold: true, Size: 18},
		{Text: "Dear Jane,", Gap: 10},
		{Text: "We are pleased to offer you the role."},
	})

	for _, marker := range [][]byte{
		[]byte("%PDF-1.4"),
		[]byte("/Type /Catalog"),
		[]byte("/Root 1 0 R"),
		[]byte("startxref"),
		[]byte("%%EOF"),
	} {
		if !bytes.Contains(out, marker) {
			t.Errorf("rendered PDF missing marker %q", marker)
		}
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Error("PDF must start with %PDF-")
	}
}

func TestRenderEscapesText(t *testing.T) {
	out := Render([]Line{{Text: "Salary (USD) 100,000 \\ net"}})
	// Parens and backslashes must be escaped inside the PDF string literal.
	if !bytes.Contains(out, []byte(`\(USD\)`)) {
		t.Error("parentheses were not escaped")
	}
}

func TestWrap(t *testing.T) {
	got := Wrap("the quick brown fox jumps", 9)
	want := []string{"the quick", "brown fox", "jumps"}
	if len(got) != len(want) {
		t.Fatalf("Wrap len = %d (%v), want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Wrap[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWrapEmpty(t *testing.T) {
	if got := Wrap("   ", 10); len(got) != 1 || got[0] != "" {
		t.Errorf("Wrap(blank) = %v, want [\"\"]", got)
	}
}
