// Package docs embeds the OpenAPI spec and serves a Swagger UI page.
package docs

import _ "embed"

// OpenAPISpec is the raw OpenAPI 3.0 document.
//
//go:embed openapi.yaml
var OpenAPISpec []byte

// SwaggerUIHTML renders Swagger UI (loaded from a CDN) against /openapi.yaml.
const SwaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Enterprise ATS — API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis]
    });
  </script>
</body>
</html>`
