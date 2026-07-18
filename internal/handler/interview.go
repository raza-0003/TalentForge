package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/middleware"
	"github.com/faizan/ats/internal/service"
)

// InterviewHandler serves interview and feedback endpoints.
type InterviewHandler struct{ svc *service.InterviewService }

// NewInterviewHandler builds an InterviewHandler.
func NewInterviewHandler(svc *service.InterviewService) *InterviewHandler {
	return &InterviewHandler{svc: svc}
}

type scheduleReq struct {
	InterviewerID   int64  `json:"interviewer_id"`
	ScheduledAt     string `json:"scheduled_at"` // RFC3339
	DurationMinutes int    `json:"duration_minutes"`
	Mode            string `json:"mode"`
	Location        string `json:"location"`
}

// Schedule books an interview for an application (:id = application id).
func (h *InterviewHandler) Schedule(c *gin.Context) {
	appID, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	var req scheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	when, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		httputil.Error(c, http.StatusBadRequest, "scheduled_at must be RFC3339 (e.g. 2026-07-20T15:00:00Z)")
		return
	}
	if req.InterviewerID <= 0 {
		httputil.Error(c, http.StatusBadRequest, "interviewer_id is required")
		return
	}
	iv, err := h.svc.Schedule(c.Request.Context(), middleware.UserID(c), appID, req.InterviewerID,
		when, req.DurationMinutes, domain.InterviewMode(req.Mode), req.Location)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.Created(c, iv)
}

// ListByApplication returns interviews for an application (:id = application id).
func (h *InterviewHandler) ListByApplication(c *gin.Context) {
	appID, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	ivs, err := h.svc.ListByApplication(c.Request.Context(), appID)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"interviews": ivs})
}

type interviewUpdateReq struct {
	ScheduledAt     string `json:"scheduled_at"`
	DurationMinutes int    `json:"duration_minutes"`
	Mode            string `json:"mode"`
	Location        string `json:"location"`
	Status          string `json:"status"`
}

// Update reschedules or changes the status of an interview (:id = interview id).
func (h *InterviewHandler) Update(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	var req interviewUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	var when time.Time
	if req.ScheduledAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			httputil.Error(c, http.StatusBadRequest, "scheduled_at must be RFC3339")
			return
		}
		when = parsed
	}
	iv, err := h.svc.Update(c.Request.Context(), id, when, req.DurationMinutes,
		domain.InterviewMode(req.Mode), req.Location, domain.InterviewStatus(req.Status))
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, iv)
}

type feedbackReq struct {
	Rating         int    `json:"rating"`
	Recommendation string `json:"recommendation"`
	Strengths      string `json:"strengths"`
	Weaknesses     string `json:"weaknesses"`
	Comments       string `json:"comments"`
}

// AddFeedback records feedback for an interview (:id = interview id).
func (h *InterviewHandler) AddFeedback(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	var req feedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	f, err := h.svc.AddFeedback(c.Request.Context(), middleware.UserID(c), id, req.Rating,
		domain.Recommendation(req.Recommendation), req.Strengths, req.Weaknesses, req.Comments)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.Created(c, f)
}

// GetFeedback returns the feedback for an interview (:id = interview id).
func (h *InterviewHandler) GetFeedback(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	f, err := h.svc.GetFeedback(c.Request.Context(), id)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, f)
}
