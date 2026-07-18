package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/auth"
	"github.com/faizan/ats/internal/domain"
	"github.com/faizan/ats/internal/middleware"
)

// Handlers bundles every HTTP handler for route registration.
type Handlers struct {
	Health      *HealthHandler
	Docs        *DocsHandler
	Auth        *AuthHandler
	Candidate   *CandidateHandler
	Job         *JobHandler
	Application *ApplicationHandler
	Interview   *InterviewHandler
	Resume      *ResumeHandler
	Search      *SearchHandler
	Offer       *OfferHandler
	Admin       *AdminHandler
}

// Register mounts all routes, applying auth and role middleware per group.
func (hs *Handlers) Register(r *gin.Engine, tm *auth.TokenManager) {
	// Liveness/readiness at the root.
	hs.Health.Register(r)

	// API docs (public).
	r.GET("/docs", hs.Docs.UI)
	r.GET("/openapi.yaml", hs.Docs.Spec)

	authmw := middleware.Auth(tm)
	recruiter := middleware.RequireRole(domain.RoleRecruiter, domain.RoleAdmin)
	candidateOnly := middleware.RequireRole(domain.RoleCandidate)
	adminOnly := middleware.RequireRole(domain.RoleAdmin)

	v1 := r.Group("/api/v1")

	// --- Public ---
	v1.POST("/auth/signup", hs.Auth.Signup)
	v1.POST("/auth/login", hs.Auth.Login)
	v1.POST("/auth/refresh", hs.Auth.Refresh)
	v1.POST("/auth/logout", hs.Auth.Logout)
	v1.GET("/jobs", hs.Job.List)
	v1.GET("/jobs/:id", hs.Job.Get)

	// --- Authenticated (any role) ---
	authed := v1.Group("")
	authed.Use(authmw)
	authed.GET("/me", hs.Auth.Me)
	authed.GET("/applications/:id", hs.Application.Get)
	authed.GET("/applications/:id/timeline", hs.Application.Timeline)
	authed.GET("/resumes/:id/download", hs.Resume.Download)
	authed.GET("/offers/:id/download", hs.Offer.Download)

	// --- Candidate ---
	cand := v1.Group("")
	cand.Use(authmw, candidateOnly)
	cand.GET("/candidate/profile", hs.Candidate.GetProfile)
	cand.PUT("/candidate/profile", hs.Candidate.UpdateProfile)
	cand.GET("/candidate/applications", hs.Candidate.MyApplications)
	cand.POST("/jobs/:id/apply", hs.Application.Apply)
	cand.POST("/candidate/resumes", hs.Resume.Upload)
	cand.GET("/candidate/resumes", hs.Resume.List)
	cand.POST("/candidate/resumes/:id/primary", hs.Resume.SetPrimary)
	cand.GET("/candidate/offers", hs.Offer.MyOffers)
	cand.POST("/offers/:id/accept", hs.Offer.Accept)
	cand.POST("/offers/:id/decline", hs.Offer.Decline)

	// --- Recruiter / Admin ---
	rec := v1.Group("")
	rec.Use(authmw, recruiter)
	rec.GET("/candidates", hs.Search.SearchCandidates)
	rec.POST("/jobs", hs.Job.Create)
	rec.PUT("/jobs/:id", hs.Job.Update)
	rec.POST("/jobs/:id/close", hs.Job.Close)
	rec.GET("/jobs/:id/applications", hs.Application.ListByJob)
	rec.PATCH("/applications/:id/status", hs.Application.ChangeStatus)
	rec.POST("/applications/:id/interviews", hs.Interview.Schedule)
	rec.GET("/applications/:id/interviews", hs.Interview.ListByApplication)
	rec.PATCH("/interviews/:id", hs.Interview.Update)
	rec.POST("/interviews/:id/feedback", hs.Interview.AddFeedback)
	rec.GET("/interviews/:id/feedback", hs.Interview.GetFeedback)
	rec.POST("/applications/:id/offers", hs.Offer.Create)
	rec.GET("/applications/:id/offers", hs.Offer.ListByApplication)
	rec.POST("/offers/:id/send", hs.Offer.Send)
	rec.POST("/offers/:id/rescind", hs.Offer.Rescind)

	// --- Admin ---
	adm := v1.Group("/admin")
	adm.Use(authmw, adminOnly)
	adm.POST("/recruiters", hs.Admin.CreateRecruiter)
	adm.GET("/recruiters", hs.Admin.ListRecruiters)
	adm.PATCH("/users/:id/active", hs.Admin.SetUserActive)
	adm.GET("/analytics", hs.Admin.Analytics)
}
