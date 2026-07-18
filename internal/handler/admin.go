package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/service"
)

// AdminHandler serves admin-only endpoints.
type AdminHandler struct{ svc *service.AdminService }

// NewAdminHandler builds an AdminHandler.
func NewAdminHandler(svc *service.AdminService) *AdminHandler { return &AdminHandler{svc: svc} }

type createRecruiterReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// CreateRecruiter provisions a recruiter account.
func (h *AdminHandler) CreateRecruiter(c *gin.Context) {
	var req createRecruiterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := h.svc.CreateRecruiter(c.Request.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.Created(c, u)
}

// ListRecruiters returns recruiter accounts.
func (h *AdminHandler) ListRecruiters(c *gin.Context) {
	page := httputil.ParsePage(c)
	users, err := h.svc.ListRecruiters(c.Request.Context(), page.Limit, page.Offset)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"recruiters": users})
}

type setActiveReq struct {
	IsActive bool `json:"is_active"`
}

// SetUserActive enables or disables a user account (:id = user id).
func (h *AdminHandler) SetUserActive(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	var req setActiveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.SetUserActive(c.Request.Context(), id, req.IsActive); err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"id": id, "is_active": req.IsActive})
}

// Analytics returns high-level hiring metrics.
func (h *AdminHandler) Analytics(c *gin.Context) {
	stats, err := h.svc.Analytics(c.Request.Context())
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, stats)
}
