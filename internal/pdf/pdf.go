// Package pdf renders very simple, single-page text PDFs from scratch with no
// external dependencies. It supports left-aligned lines in regular or bold
// Helvetica at a chosen size — enough for a one-page offer letter.
package pdf

import (
	"bytes"
	"fmt"
	"strings"
)

// Line is one line of text on the page.
type Line struct {
	Text string
	Bold bool
	Size float64 // points; defaults to 11
	Gap  float64 // extra vertical space (points) before this line
}

const (
	pageWidth  = 612.0 // US Letter, points
	pageHeight = 792.0
	marginLeft = 72.0
	marginTop  = 72.0
)

// Render returns the bytes of a one-page PDF containing the given lines,
// top-to-bottom. Lines that overflow the page are clipped.
func Render(lines []Line) []byte {
	var content bytes.Buffer
	content.WriteString("BT\n")
	y := pageHeight - marginTop
	for _, ln := range lines {
		size := ln.Size
		if size <= 0 {
			size = 11
		}
		y -= ln.Gap + size*1.35
		if y < marginTop {
			break
		}
		font := "/F1"
		if ln.Bold {
			font = "/F2"
		}
		fmt.Fprintf(&content, "%s %.1f Tf\n", font, size)
		fmt.Fprintf(&content, "1 0 0 1 %.1f %.1f Tm\n", marginLeft, y)
		fmt.Fprintf(&content, "(%s) Tj\n", escape(ln.Text))
	}
	content.WriteString("ET")

	// Object bodies, 1-indexed by position. Offsets are computed as bytes are
	// written so the cross-reference table is always correct.
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] "+
			"/Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >>", pageWidth, pageHeight),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", content.Len(), content.String()),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, body := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}

	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, xref)
	return buf.Bytes()
}

// Wrap splits text into lines of at most max characters on word boundaries.
func Wrap(text string, max int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0)
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) <= max {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	return append(lines, cur)
}

// escape makes text safe inside a PDF string literal.
func escape(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`(`, `\(`,
		`)`, `\)`,
		"\r", "",
		"\n", " ",
	).Replace(s)
}
