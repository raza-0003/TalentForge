package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/httputil"
	"github.com/faizan/ats/internal/middleware"
	"github.com/faizan/ats/internal/service"
)

// OfferHandler serves offer-letter endpoints.
type OfferHandler struct{ svc *service.OfferService }

// NewOfferHandler builds an OfferHandler.
func NewOfferHandler(svc *service.OfferService) *OfferHandler { return &OfferHandler{svc: svc} }

type offerReq struct {
	PositionTitle  string   `json:"position_title"`
	SalaryAmount   *float64 `json:"salary_amount"`
	SalaryCurrency string   `json:"salary_currency"`
	StartDate      string   `json:"start_date"` // YYYY-MM-DD
	ExpiresAt      string   `json:"expires_at"` // RFC3339
}

// Create drafts an offer and renders its PDF (:id = application id).
func (h *OfferHandler) Create(c *gin.Context) {
	appID, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	var req offerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	in := service.OfferInput{
		PositionTitle:  req.PositionTitle,
		SalaryAmount:   req.SalaryAmount,
		SalaryCurrency: req.SalaryCurrency,
	}
	if req.StartDate != "" {
		d, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			httputil.Error(c, http.StatusBadRequest, "start_date must be YYYY-MM-DD")
			return
		}
		in.StartDate = &d
	}
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			httputil.Error(c, http.StatusBadRequest, "expires_at must be RFC3339")
			return
		}
		in.ExpiresAt = &t
	}

	offer, err := h.svc.Create(c.Request.Context(), middleware.UserID(c), appID, in)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.Created(c, offer)
}

// ListByApplication lists an application's offers (:id = application id).
func (h *OfferHandler) ListByApplication(c *gin.Context) {
	appID, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	offers, err := h.svc.ListByApplication(c.Request.Context(), appID)
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"offers": offers})
}

// MyOffers lists the current candidate's offers.
func (h *OfferHandler) MyOffers(c *gin.Context) {
	offers, err := h.svc.ListByCandidate(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, gin.H{"offers": offers})
}

// Send delivers a draft offer (:id = offer id).
func (h *OfferHandler) Send(c *gin.Context) {
	h.transition(c, func(id, actor int64) (any, error) {
		return h.svc.Send(c.Request.Context(), actor, id)
	})
}

// Rescind withdraws an offer (:id = offer id).
func (h *OfferHandler) Rescind(c *gin.Context) {
	h.transition(c, func(id, actor int64) (any, error) {
		return h.svc.Rescind(c.Request.Context(), actor, id)
	})
}

// Accept records a candidate's acceptance (:id = offer id).
func (h *OfferHandler) Accept(c *gin.Context) {
	h.transition(c, func(id, actor int64) (any, error) {
		return h.svc.Respond(c.Request.Context(), actor, id, true)
	})
}

// Decline records a candidate's decline (:id = offer id).
func (h *OfferHandler) Decline(c *gin.Context) {
	h.transition(c, func(id, actor int64) (any, error) {
		return h.svc.Respond(c.Request.Context(), actor, id, false)
	})
}

func (h *OfferHandler) transition(c *gin.Context, fn func(id, actor int64) (any, error)) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	res, err := fn(id, middleware.UserID(c))
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	httputil.OK(c, res)
}

// Download streams an offer PDF (:id = offer id). Owner candidate or recruiter/admin.
func (h *OfferHandler) Download(c *gin.Context) {
	id, ok := httputil.ParseIDParam(c, "id")
	if !ok {
		return
	}
	rc, offer, err := h.svc.OpenForDownload(c.Request.Context(), id, middleware.UserID(c), middleware.CurrentRole(c))
	if err != nil {
		httputil.FromDomain(c, err)
		return
	}
	defer rc.Close()
	c.Header("Content-Disposition", `attachment; filename="offer-`+strconv.FormatInt(offer.ID, 10)+`.pdf"`)
	c.DataFromReader(http.StatusOK, -1, "application/pdf", rc, nil)
}
