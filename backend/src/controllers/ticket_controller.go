package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"simplehelp/src/middleware"
	"simplehelp/src/models"
	"simplehelp/src/repositories"
	"simplehelp/src/services"

	"github.com/gin-gonic/gin"
)

// TicketController wires ticket HTTP endpoints to the TicketService.
type TicketController struct {
	tickets *services.TicketService
	users   repositories.UserRepository
}

// NewTicketController builds a TicketController.
func NewTicketController(tickets *services.TicketService, users repositories.UserRepository) *TicketController {
	return &TicketController{tickets: tickets, users: users}
}

type createTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
}

// Create handles POST /api/tickets.
func (c *TicketController) Create(ctx *gin.Context) {
	var req createTicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_json", map[string]string{
			"body": "could not parse JSON body",
		})
		return
	}
	uid := middleware.UserIDFrom(ctx)
	t, err := c.tickets.Create(ctx.Request.Context(), services.CreateTicketInput{
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Priority:    req.Priority,
		RequesterID: uid,
	})
	if err != nil {
		c.writeTicketError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, t)
}

// Get handles GET /api/tickets/{id}.
func (c *TicketController) Get(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}
	detail, err := c.tickets.Get(ctx.Request.Context(), id)
	if err != nil {
		c.writeTicketError(ctx, err)
		return
	}
	actor := c.actorFromContext(ctx)
	if !services.CanView(actor, detail.Ticket) {
		writeError(ctx, http.StatusForbidden, "forbidden", nil)
		return
	}
	ctx.JSON(http.StatusOK, detail)
}

// List handles GET /api/tickets.
func (c *TicketController) List(ctx *gin.Context) {
	uid := middleware.UserIDFrom(ctx)
	role := middleware.RoleFrom(ctx)
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	tickets, err := c.tickets.List(ctx.Request.Context(), uid, role, services.ListFilters{
		Limit: limit, Offset: offset,
	})
	if err != nil {
		c.writeTicketError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"tickets": tickets})
}

type updateTicketRequest struct {
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	AssigneeID *int64 `json:"assignee_id"`
}

// Update handles PATCH /api/tickets/{id}.
func (c *TicketController) Update(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}
	var req updateTicketRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_json", map[string]string{
			"body": "could not parse JSON body",
		})
		return
	}
	actor := c.actorFromContext(ctx)
	t, err := c.tickets.Update(ctx.Request.Context(), id, services.UpdateTicketInput{
		Status: req.Status, Priority: req.Priority, AssigneeID: req.AssigneeID,
	}, actor)
	if err != nil {
		c.writeTicketError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, t)
}

type createCommentRequest struct {
	Body       string `json:"body"`
	IsInternal bool   `json:"is_internal"`
}

// CreateComment handles POST /api/tickets/{id}/comments.
func (c *TicketController) CreateComment(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}
	var req createCommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeError(ctx, http.StatusBadRequest, "invalid_json", map[string]string{
			"body": "could not parse JSON body",
		})
		return
	}
	actor := c.actorFromContext(ctx)
	// Verify visibility first.
	detail, err := c.tickets.Get(ctx.Request.Context(), id)
	if err != nil {
		c.writeTicketError(ctx, err)
		return
	}
	if !services.CanView(actor, detail.Ticket) {
		writeError(ctx, http.StatusForbidden, "forbidden", nil)
		return
	}
	comment, err := c.tickets.CreateComment(ctx.Request.Context(), services.CreateCommentInput{
		TicketID: id, AuthorID: actor.ID, Body: req.Body, IsInternal: req.IsInternal,
	}, actor)
	if err != nil {
		c.writeTicketError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, comment)
}

// actorFromContext loads the full user from the ID in the JWT. Used for
// role/visibility checks in the service layer.
func (c *TicketController) actorFromContext(ctx *gin.Context) *models.User {
	uid := middleware.UserIDFrom(ctx)
	u, err := c.users.GetByID(ctx.Request.Context(), uid)
	if err != nil {
		return &models.User{ID: uid, Role: middleware.RoleFrom(ctx)}
	}
	return u
}

// parseID extracts {id} as int64. Writes a 400 on failure.
func parseID(ctx *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(ctx, http.StatusBadRequest, "invalid_id", map[string]string{
			"id": "must be a positive integer",
		})
		return 0, false
	}
	return id, true
}

// writeTicketError maps service errors to HTTP responses.
func (c *TicketController) writeTicketError(ctx *gin.Context, err error) {
	var ve *services.ValidationError
	if errors.As(err, &ve) {
		writeError(ctx, http.StatusBadRequest, "validation_failed", ve.Fields)
		return
	}
	switch {
	case errors.Is(err, services.ErrNotFound):
		writeError(ctx, http.StatusNotFound, "not_found", nil)
	case errors.Is(err, services.ErrInvalidTransition):
		writeError(ctx, http.StatusUnprocessableEntity, "invalid_transition", nil)
	case errors.Is(err, services.ErrForbidden):
		writeError(ctx, http.StatusForbidden, "forbidden", nil)
	default:
		slog.Error("ticket op failed", "error", err)
		writeError(ctx, http.StatusInternalServerError, "internal_error", nil)
	}
}
