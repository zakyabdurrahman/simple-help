package controllers

import (
	"errors"
	"log/slog"
	"net/http"

	"simplehelp/src/services"

	"github.com/gin-gonic/gin"
)

// AuthController wires auth-related HTTP endpoints to the AuthService.
type AuthController struct {
	auth *services.AuthService
}

// NewAuthController builds an AuthController.
func NewAuthController(auth *services.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

// registerRequest is the JSON body for POST /api/auth/register.
type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles POST /api/auth/register.
func (c *AuthController) Register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_json", map[string]string{
			"body": "could not parse JSON body",
		})
		return
	}

	user, err := c.auth.Register(ctx.Request.Context(), services.RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.writeRegisterError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

// writeRegisterError maps a service error to the appropriate HTTP response.
func (c *AuthController) writeRegisterError(ctx *gin.Context, err error) {
	var ve *services.ValidationError
	if errors.As(err, &ve) {
		writeError(ctx, http.StatusBadRequest, "validation_failed", ve.Fields)
		return
	}
	if errors.Is(err, services.ErrEmailExists) {
		writeError(ctx, http.StatusConflict, "email_exists", map[string]string{
			"email": "already in use",
		})
		return
	}
	slog.Error("register failed unexpectedly", "error", err)
	writeError(ctx, http.StatusInternalServerError, "internal_error", nil)
}
