package controllers

import "github.com/gin-gonic/gin"

// errorResponse is the standard error envelope returned by the API.
// Fields is optional and used for validation errors.
type errorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// writeError writes a JSON error response with the given status and message.
// fields may be nil.
func writeError(ctx *gin.Context, status int, code string, fields map[string]string) {
	ctx.JSON(status, errorResponse{Error: code, Fields: fields})
}
