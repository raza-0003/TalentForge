package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/middleware"
	"github.com/faizan/ats/internal/service"
)

// JobHandler serves job posting endpoints.
type JobHandler struct{ svc *service.JobService }

// NewJobHandler builds a JobHandler.
func NewJobHandler(svc *service.JobService) *JobHandler { return &JobHandler{svc: svc} }

// List returns jobs, filterable by ?status and ?q (search).
func (h *JobHandler) List(c *gin.Context) {
	page := httputil.ParsePage(c)
	status := domain.JobStatus(c.Query("status"))
	jobs, err := h.svc.List(c.Request.Context(), status, c.Query("q"), page.Limit, page.Offset)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"jobs": jobs})
}

// Get returns a single job by id.
func (h *JobHandler) Get(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	job, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, job)
}

type jobReq struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Department     string `json:"department"`
	Location       string `json:"location"`
	EmploymentType string `json:"employment_type"`
	MinExperience  int    `json:"min_experience"`
	Status         string `json:"status"`
}

func (r jobReq) toDomain() *domain.Job {
	return &domain.Job{
		Title:          r.Title,
		Description:    r.Description,
		Department:     r.Department,
		Location:       r.Location,
		EmploymentType: r.EmploymentType,
		MinExperience:  r.MinExperience,
		Status:         domain.JobStatus(r.Status),
	}
}

// Create posts a new job (recruiter/admin).
func (h *JobHandler) Create(c *gin.Context) {
	var req jobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	job := req.toDomain()
	if err := h.svc.Create(c.Request.Context(), middleware.UserID(c), job); err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.Created(c, job)
}

// Update edits a job (owner recruiter/admin).
func (h *JobHandler) Update(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	var req jobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	job := req.toDomain()
	job.ID = id
	if err := h.svc.Update(c.Request.Context(), middleware.UserID(c), middleware.CurrentRole(c), job); err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, job)
}

// Close marks a job closed (owner recruiter/admin).
func (h *JobHandler) Close(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Close(c.Request.Context(), middleware.UserID(c), middleware.CurrentRole(c), id); err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"status": "closed"})
}
