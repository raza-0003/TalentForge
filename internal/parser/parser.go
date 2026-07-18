// Package parser extracts plain text and structured fields (emails, phones,
// skills) from resume files. Text and DOCX are handled with the standard
// library; other types (e.g. PDF) are stored but not text-extracted here.
package parser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Fields are the structured values pulled from resume text.
type Fields struct {
	Emails []string
	Phones []string
	Skills []string
}

// ParsedResume is the JSON persisted to resumes.parsed_data.
type ParsedResume struct {
	Emails    []string `json:"emails"`
	Phones    []string `json:"phones"`
	Skills    []string `json:"skills"`
	TextChars int      `json:"text_chars"`
	Extractor string   `json:"extractor"` // text | docx | none
	Note      string   `json:"note,omitempty"`
}

// Parse extracts text and fields from a resume file.
func Parse(contentType, fileName string, data []byte) (ParsedResume, error) {
	text, method, err := ExtractText(contentType, fileName, data)
	if err != nil {
		return ParsedResume{}, err
	}
	f := ExtractFields(text)
	pr := ParsedResume{
		Emails:    f.Emails,
		Phones:    f.Phones,
		Skills:    f.Skills,
		TextChars: len([]rune(text)),
		Extractor: method,
	}
	switch method {
	case "none":
		pr.Note = "text extraction not supported for this file type"
	case "pdf_unreadable":
		pr.Note = "could not extract text from this PDF (it may be scanned/image-based or encrypted)"
	}
	return pr, nil
}

// ExtractText returns plain text plus the extractor used.
func ExtractText(contentType, fileName string, data []byte) (string, string, error) {
	name := strings.ToLower(fileName)
	switch {
	case strings.HasSuffix(name, ".docx") || strings.Contains(contentType, "wordprocessingml"):
		t, err := extractDocx(data)
		if err != nil {
			return "", "docx", err
		}
		return t, "docx", nil
	case strings.HasSuffix(name, ".pdf") || strings.Contains(contentType, "pdf"):
		if t, ok := extractPDF(data); ok {
			return t, "pdf", nil
		}
		// Scanned, image-only, or encrypted PDFs yield no text — store the file
		// but record that extraction was unreadable (not a task failure).
		return "", "pdf_unreadable", nil
	case strings.HasPrefix(contentType, "text/") ||
		strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".rtf"):
		return string(data), "text", nil
	default:
		return "", "none", nil
	}
}

// extractPDF returns the plain text of a PDF. It is panic-safe: the underlying
// library can panic on malformed input, so a failure returns ok=false rather
// than crashing the worker.
func extractPDF(data []byte) (text string, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			text, ok = "", false
		}
	}()
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", false
	}
	r, err := reader.GetPlainText()
	if err != nil {
		return "", false
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return "", false
	}
	return buf.String(), true
}

var tagRe = regexp.MustCompile(`<[^>]+>`)

// extractDocx reads word/document.xml from the .docx zip and strips XML tags.
func extractDocx(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	for _, file := range zr.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("open document.xml: %w", err)
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return "", fmt.Errorf("read document.xml: %w", err)
		}
		s := string(raw)
		s = strings.ReplaceAll(s, "</w:p>", "\n")
		s = strings.ReplaceAll(s, "<w:br/>", "\n")
		s = strings.ReplaceAll(s, "<w:br />", "\n")
		s = tagRe.ReplaceAllString(s, "")
		return html.UnescapeString(s), nil
	}
	return "", nil
}

var (
	emailRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phoneRe = regexp.MustCompile(`\+?\d[\d\s\-().]{7,}\d`)
)

// ExtractFields pulls emails, phones, and skills from text.
func ExtractFields(text string) Fields {
	return Fields{
		Emails: capSlice(dedupe(emailRe.FindAllString(text, -1)), 5),
		Phones: capSlice(dedupe(cleanSpaces(phoneRe.FindAllString(text, 10))), 5),
		Skills: matchSkills(text),
	}
}

type skillPattern struct {
	name string
	re   *regexp.Regexp
}

// SkillNames is the known-skill taxonomy used both to extract skills from resume
// text and to canonicalize free-text skills for search.
var SkillNames = []string{
	"Go", "Golang", "Python", "Java", "JavaScript", "TypeScript", "C++", "C#", "Ruby", "PHP", "Rust", "Kotlin", "Swift", "Scala",
	"React", "Angular", "Vue", "Node.js", "Django", "Flask", "Spring", "Gin", "Express", ".NET",
	"PostgreSQL", "MySQL", "MongoDB", "Redis", "SQLite", "Elasticsearch", "Cassandra",
	"Docker", "Kubernetes", "AWS", "Azure", "GCP", "Terraform", "Ansible", "Linux",
	"Git", "CI/CD", "GraphQL", "REST", "gRPC", "Kafka", "RabbitMQ", "Nginx",
	"HTML", "CSS", "SQL", "Bash", "Machine Learning", "TensorFlow", "PyTorch",
}

// Skill names are matched case-sensitively in resume text (resumes almost always
// capitalize technology names correctly), which avoids false positives like
// "go" -> "Go".
var skillPatterns = buildSkillPatterns(SkillNames)

// canonicalByLower maps a lowercased skill to its canonical spelling.
var canonicalByLower = buildCanonical(SkillNames)

func buildCanonical(names []string) map[string]string {
	m := make(map[string]string, len(names))
	for _, n := range names {
		m[strings.ToLower(n)] = n
	}
	return m
}

// Canonicalize maps known skills to their canonical spelling (case-insensitive)
// and de-duplicates; unknown skills are kept trimmed as entered. Applied both
// when a candidate saves their profile and to search terms, so casing
// differences ("go" vs "Go") still match under the exact array operators.
func Canonicalize(skills []string) []string {
	out := make([]string, 0, len(skills))
	seen := map[string]bool{}
	for _, s := range skills {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if canon, ok := canonicalByLower[strings.ToLower(s)]; ok {
			s = canon
		}
		key := strings.ToLower(s)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	return out
}

func buildSkillPatterns(names []string) []skillPattern {
	out := make([]skillPattern, 0, len(names))
	for _, n := range names {
		// Require a non-alphanumeric boundary on each side so "Java" doesn't
		// match inside "JavaScript"; the keyword's own punctuation (C++, .NET)
		// is part of the literal.
		re := regexp.MustCompile(`(^|[^a-zA-Z0-9])` + regexp.QuoteMeta(n) + `($|[^a-zA-Z0-9])`)
		out = append(out, skillPattern{name: n, re: re})
	}
	return out
}

func matchSkills(text string) []string {
	out := []string{}
	for _, s := range skillPatterns {
		if s.re.MatchString(text) {
			out = append(out, s.name)
		}
	}
	return out
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func cleanSpaces(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, strings.Join(strings.Fields(s), " "))
	}
	return out
}

func capSlice(in []string, n int) []string {
	if len(in) > n {
		return in[:n]
	}
	return in
}
