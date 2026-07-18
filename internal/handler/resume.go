package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/middleware"
	"github.com/faizan/ats/internal/service"
)

// ResumeHandler serves resume upload and retrieval endpoints.
type ResumeHandler struct{ svc *service.ResumeService }

// NewResumeHandler builds a ResumeHandler.
func NewResumeHandler(svc *service.ResumeService) *ResumeHandler { return &ResumeHandler{svc: svc} }

// Upload handles a multipart resume upload: form field "file", optional
// "primary=true".
func (h *ResumeHandler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		httputil.Error(c, http.StatusBadRequest, "missing 'file' in multipart form")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		httputil.Error(c, http.StatusBadRequest, "could not read the uploaded file")
		return
	}
	defer f.Close()

	res, err := h.svc.Upload(c.Request.Context(), middleware.UserID(c),
		fileHeader.Filename, fileHeader.Header.Get("Content-Type"), fileHeader.Size,
		f, c.PostForm("primary") == "true")
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.Created(c, res)
}

// List returns the current candidate's resumes.
func (h *ResumeHandler) List(c *gin.Context) {
	resumes, err := h.svc.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"resumes": resumes})
}

// SetPrimary marks a resume as primary (:id = resume id).
func (h *ResumeHandler) SetPrimary(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.SetPrimary(c.Request.Context(), middleware.UserID(c), id); err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"id": id, "is_primary": true})
}

// Download streams a resume file. The owning candidate or any recruiter/admin
// may download it (:id = resume id).
func (h *ResumeHandler) Download(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	res, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	if middleware.CurrentRole(c) == domain.RoleCandidate && res.CandidateUserID != middleware.UserID(c) {
		httputil.Error(c, http.StatusForbidden, "forbidden")
		return
	}
	rc, err := h.svc.Open(c.Request.Context(), res)
	if err != nil {
		httputil.Error(c, http.StatusNotFound, "file not found")
		return
	}
	defer rc.Close()

	ctype := res.ContentType
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	c.Header("Content-Disposition", `attachment; filename="`+res.FileName+`"`)
	c.DataFromReader(http.StatusOK, res.SizeBytes, ctype, rc, nil)
}
