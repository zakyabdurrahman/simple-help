package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"simplehelp/src/services"

	"github.com/gin-gonic/gin"
)

// UserController wires admin user-management endpoints.
type UserController struct {
	users *services.UserService
}

// NewUserController builds a UserController.
func NewUserController(users *services.UserService) *UserController {
	return &UserController{users: users}
}

// List handles GET /api/users (admin only — enforced by route middleware).
func (c *UserController) List(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	users, err := c.users.List(ctx.Request.Context(), limit, offset)
	if err != nil {
		slog.Error("user list failed", "error", err)
		writeError(ctx, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"users": users})
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

// UpdateRole handles PATCH /api/users/{id}/role.
func (c *UserController) UpdateRole(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}
	var req updateRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_json", map[string]string{
			"body": "could not parse JSON body",
		})
		return
	}
	user, err := c.users.UpdateRole(ctx.Request.Context(), id, req.Role)
	if err != nil {
		var ve *services.ValidationError
		if errors.As(err, &ve) {
			writeError(ctx, http.StatusBadRequest, "validation_failed", ve.Fields)
			return
		}
		if errors.Is(err, services.ErrNotFound) {
			writeError(ctx, http.StatusNotFound, "not_found", nil)
			return
		}
		slog.Error("role update failed", "error", err)
		writeError(ctx, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	ctx.JSON(http.StatusOK, user)
}
