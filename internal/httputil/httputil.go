// Package httputil provides small HTTP response helpers shared by handlers.
package httputil

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/domain"
)

// OK writes a 200 with data.
func OK(c *gin.Context, data any) { c.JSON(http.StatusOK, data) }

// Created writes a 201 with data.
func Created(c *gin.Context, data any) { c.JSON(http.StatusCreated, data) }

// Error aborts with a JSON error body.
func Error(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

// FromDomain maps a domain sentinel error to an HTTP response.
func FromDomain(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		Error(c, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrConflict):
		Error(c, http.StatusConflict, "already exists")
	case errors.Is(err, domain.ErrInvalidCredentials):
		Error(c, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, domain.ErrForbidden):
		Error(c, http.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrValidation):
		Error(c, http.StatusBadRequest, err.Error())
	default:
		Error(c, http.StatusInternalServerError, "internal server error")
	}
}

// Page holds pagination parameters.
type Page struct {
	Limit  int
	Offset int
}

// ParsePage reads ?limit & ?offset with sane defaults and bounds.
func ParsePage(c *gin.Context) Page {
	p := Page{Limit: 20, Offset: 0}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			p.Limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = n
		}
	}
	return p
}

// ParseIDParam reads a positive int64 path parameter (e.g. :id).
func ParseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		Error(c, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}
