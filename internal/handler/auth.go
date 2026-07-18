package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/middleware"
	"github.com/faizan/ats/internal/service"
)

// AuthHandler serves authentication endpoints.
type AuthHandler struct{ svc *service.AuthService }

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

type signupReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// Signup registers a new candidate.
func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := h.svc.Signup(c.Request.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.Created(c, u)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates a user and returns a token pair.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	u, pair, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"user": u, "tokens": pair})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh rotates a refresh token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		httputil.Error(c, http.StatusBadRequest, "refresh_token is required")
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, pair)
}

// Logout revokes a refresh token.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		httputil.Error(c, http.StatusBadRequest, "refresh_token is required")
		return
	}
	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httputil.FromDomain(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Me returns the authenticated user.
func (h *AuthHandler) Me(c *gin.Context) {
	u, err := h.svc.Me(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, u)
}
