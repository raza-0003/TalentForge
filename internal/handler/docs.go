package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/faizan/ats/internal/docs"
)

// DocsHandler serves the OpenAPI spec and Swagger UI.
type DocsHandler struct{}

// NewDocsHandler builds a DocsHandler.
func NewDocsHandler() *DocsHandler { return &DocsHandler{} }

// Spec serves the raw OpenAPI YAML.
func (h *DocsHandler) Spec(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.OpenAPISpec)
}

// UI serves the Swagger UI page.
func (h *DocsHandler) UI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs.SwaggerUIHTML))
}
