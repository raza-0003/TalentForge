package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/service"
)

// SearchHandler serves recruiter candidate search.
type SearchHandler struct{ candidates *service.CandidateService }

// NewSearchHandler builds a SearchHandler.
func NewSearchHandler(candidates *service.CandidateService) *SearchHandler {
	return &SearchHandler{candidates: candidates}
}

// SearchCandidates finds candidates by skills and/or name (recruiter/admin).
// Query params: ?skills=Go,PostgreSQL&match=all|any&q=<name>&limit=&offset=
func (h *SearchHandler) SearchCandidates(c *gin.Context) {
	var skills []string
	if raw := c.Query("skills"); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			if s = strings.TrimSpace(s); s != "" {
				skills = append(skills, s)
			}
		}
	}
	matchAll := c.Query("match") == "all"
	page := httputil.ParsePage(c)

	results, err := h.candidates.Search(c.Request.Context(), skills, c.Query("q"), matchAll, page.Limit, page.Offset)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"candidates": results})
}
