package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/middleware"
	"github.com/faizan/ats/internal/service"
)

// CandidateHandler serves candidate profile and application endpoints.
type CandidateHandler struct {
	profiles *service.CandidateService
	apps     *service.ApplicationService
}

// NewCandidateHandler builds a CandidateHandler.
func NewCandidateHandler(profiles *service.CandidateService, apps *service.ApplicationService) *CandidateHandler {
	return &CandidateHandler{profiles: profiles, apps: apps}
}

// GetProfile returns the current candidate's profile.
func (h *CandidateHandler) GetProfile(c *gin.Context) {
	p, err := h.profiles.GetProfile(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, p)
}

type profileReq struct {
	Phone    string            `json:"phone"`
	Headline string            `json:"headline"`
	Location string            `json:"location"`
	Links    map[string]string `json:"links"`
	Skills   []string          `json:"skills"`
}

// UpdateProfile creates or updates the current candidate's profile.
func (h *CandidateHandler) UpdateProfile(c *gin.Context) {
	var req profileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	p := &domain.CandidateProfile{
		UserID:   middleware.UserID(c),
		Phone:    req.Phone,
		Headline: req.Headline,
		Location: req.Location,
		Links:    req.Links,
		Skills:   req.Skills,
	}
	if err := h.profiles.UpdateProfile(c.Request.Context(), p); err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, p)
}

// MyApplications lists the current candidate's applications.
func (h *CandidateHandler) MyApplications(c *gin.Context) {
	page := httputil.ParsePage(c)
	apps, err := h.apps.ListByCandidate(c.Request.Context(), middleware.UserID(c), page.Limit, page.Offset)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"applications": apps})
}
