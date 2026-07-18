// Package domain holds the core entities and sentinel errors, with no
// dependencies on any other internal package.
package domain

import "errors"

// Sentinel errors. Repositories and services return these; the HTTP layer maps
// them to status codes (see internal/httputil).
var (
	ErrNotFound           = errors.New("resource not found")
	ErrConflict           = errors.New("resource already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrForbidden          = errors.New("forbidden")
	ErrValidation         = errors.New("validation failed")
)
