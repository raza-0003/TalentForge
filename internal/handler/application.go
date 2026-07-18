package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/middleware"
	"github.com/faizan/ats/internal/service"
)

// ApplicationHandler serves application endpoints.
type ApplicationHandler struct{ svc *service.ApplicationService }

// NewApplicationHandler builds an ApplicationHandler.
func NewApplicationHandler(svc *service.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{svc: svc}
}

type applyReq struct {
	ResumeID    *int64 `json:"resume_id"`
	CoverLetter string `json:"cover_letter"`
}

// Apply submits the current candidate's application to a job (:id = job id).
func (h *ApplicationHandler) Apply(c *gin.Context) {
	jobID, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	var req applyReq
	_ = c.ShouldBindJSON(&req) // body is optional (resume + cover letter)
	app, err := h.svc.Apply(c.Request.Context(), middleware.UserID(c), jobID, req.ResumeID, req.CoverLetter)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.Created(c, app)
}

// ListByJob lists applications for a job (:id = job id), filterable by ?status.
func (h *ApplicationHandler) ListByJob(c *gin.Context) {
	jobID, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	page := httputil.ParsePage(c)
	status := domain.ApplicationStatus(c.Query("status"))
	apps, err := h.svc.ListByJob(c.Request.Context(), jobID, status, page.Limit, page.Offset)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"applications": apps})
}

// Get returns a single application. Candidates may only view their own.
func (h *ApplicationHandler) Get(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	app, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	if middleware.CurrentRole(c) == domain.RoleCandidate && app.CandidateID != middleware.UserID(c) {
		httputil.Error(c, http.StatusForbidden, "forbidden")
		return
	}
	httputil.OK(c, app)
}

// Timeline returns an application's event history (owner candidate or recruiter/admin).
func (h *ApplicationHandler) Timeline(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	app, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	if middleware.CurrentRole(c) == domain.RoleCandidate && app.CandidateID != middleware.UserID(c) {
		httputil.Error(c, http.StatusForbidden, "forbidden")
		return
	}
	events, err := h.svc.Timeline(c.Request.Context(), id)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"events": events})
}

type statusReq struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

// ChangeStatus advances an application (:id = application id) (recruiter/admin).
func (h *ApplicationHandler) ChangeStatus(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	var req statusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	app, err := h.svc.ChangeStatus(c.Request.Context(), middleware.UserID(c), id,
		domain.ApplicationStatus(req.Status), req.Note)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, app)
}
