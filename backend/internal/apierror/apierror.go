// Package apierror provides the one consistent error response shape used
// across every handler in the API (section 34), so M6 onward reuses this
// rather than each domain inventing its own error JSON.
package apierror

import "github.com/gin-gonic/gin"

type body struct {
	Error detail `json:"error"`
}

type detail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Respond writes a structured error response and nothing else — it does
// not call c.Abort(); callers decide whether to abort the chain.
// `details` is omitted from the JSON entirely when nil, so handlers that
// have nothing more specific to say don't need an empty "details": {}.
func Respond(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, body{Error: detail{Code: code, Message: message, Details: details}})
}
