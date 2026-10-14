// Package api embeds the OpenAPI document served by the HTTP API.
package api

import _ "embed"

// OpenAPI is the OpenAPI 3 document for the implemented routes.
//
//go:embed openapi.json
var OpenAPI []byte
