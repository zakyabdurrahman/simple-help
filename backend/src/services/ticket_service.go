package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"simplehelp/src/models"
	"simplehelp/src/repositories"
	"simplehelp/src/validation"
)

// TicketService contains the business logic for ticket flows.
type TicketService struct {
	tickets  repositories.TicketRepository
	comments repositories.CommentRepository
}

// NewTicketService builds a TicketService.
func NewTicketService(tickets repositories.TicketRepository, comments repositories.CommentRepository) *TicketService {
	return &TicketService{tickets: tickets, comments: comments}
}

// CreateTicketInput is the validated payload for creating a ticket.
type CreateTicketInput struct {
	Title       string
	Description string
	Category    string
	Priority    string
	RequesterID int64
}

// Create creates a ticket owned by the requester. Defaults: priority medium,
// status open.
func (s *TicketService) Create(ctx context.Context, in CreateTicketInput) (*models.Ticket, error) {
	errs := validation.FieldErrors{}
	if in.Title == "" {
		errs["title"] = "is required"
	} else if len(in.Title) > 200 {
		errs["title"] = "must be 200 characters or fewer"
	}
	if in.Description == "" {
		errs["description"] = "is required"
	}
	priority := in.Priority
	if priority == "" {
		priority = models.PriorityMedium
	}
	if priority != models.PriorityLow && priority != models.PriorityMedium && priority != models.PriorityHigh {
		errs["priority"] = "must be one of low, medium, high"
	}
	if !errs.IsEmpty() {
		return nil, &ValidationError{Fields: errs}
	}

	t := &models.Ticket{
		Title:       in.Title,
		Description: in.Description,
		Priority:    priority,
		RequesterID: in.RequesterID,
	}
	if in.Category != "" {
		t.Category = sql.NullString{String: in.Category, Valid: true}
	}
	return s.tickets.Create(ctx, t)
}

// TicketDetail is a ticket plus its comments, returned by GET /api/tickets/{id}.
type TicketDetail struct {
	Ticket   *models.Ticket
	Comments []*models.Comment
}

// ticketDetailJSON is the JSON shape of TicketDetail. Ticket fields are
// inlined (no "ticket" wrapper) and comments is a top-level array, matching
// the API contract in api.md.
type ticketDetailJSON struct {
	ID          int64              `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Category    any                `json:"category"`
	Priority    string             `json:"priority"`
	Status      string             `json:"status"`
	RequesterID int64              `json:"requester_id"`
	AssigneeID  any                `json:"assignee_id"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
	Comments    []*models.Comment  `json:"comments"`
}

// MarshalJSON renders TicketDetail with the ticket fields inlined plus a
// comments array.
func (d TicketDetail) MarshalJSON() ([]byte, error) {
	if d.Ticket == nil {
		return []byte("null"), nil
	}
	out := ticketDetailJSON{
		ID: d.Ticket.ID, Title: d.Ticket.Title, Description: d.Ticket.Description,
		Priority: d.Ticket.Priority, Status: d.Ticket.Status,
		RequesterID: d.Ticket.RequesterID,
		CreatedAt:   d.Ticket.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   d.Ticket.UpdatedAt.Format(time.RFC3339Nano),
		Comments:    d.Comments,
	}
	if d.Ticket.Category.Valid {
		out.Category = d.Ticket.Category.String
	}
	if d.Ticket.AssigneeID.Valid {
		out.AssigneeID = d.Ticket.AssigneeID.Int64
	}
	return json.Marshal(out)
}

// Get returns a ticket and its comments. Visibility is enforced by the
// controller via CanView; this method assumes the caller is permitted.
func (s *TicketService) Get(ctx context.Context, id int64) (*TicketDetail, error) {
	t, err := s.tickets.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("ticketService.Get: %w", err)
	}
	comments, err := s.comments.ListByTicket(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ticketService.Get comments: %w", err)
	}
	return &TicketDetail{Ticket: t, Comments: comments}, nil
}

// ListFilters are the query params for GET /api/tickets.
type ListFilters struct {
	Limit  int
	Offset int
}

// List returns tickets visible to the user. Requesters see only their own;
// agents/admins see all.
func (s *TicketService) List(ctx context.Context, userID int64, role string, f ListFilters) ([]*models.Ticket, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return s.tickets.List(ctx, userID, role, f.Limit, f.Offset)
}

// CanView reports whether a user may view a given ticket.
func CanView(user *models.User, t *models.Ticket) bool {
	if user == nil || t == nil {
		return false
	}
	switch user.Role {
	case models.RoleAdmin, models.RoleAgent:
		return true
	case models.RoleRequester:
		return t.RequesterID == user.ID
	}
	return false
}

// transitionTable maps (from, to) -> roles allowed to make that transition.
var transitionTable = map[string]map[string][]string{
	models.StatusOpen: {
		models.StatusInProgress: {models.RoleAgent, models.RoleAdmin},
		models.StatusClosed:     {models.RoleRequester, models.RoleAdmin},
	},
	models.StatusInProgress: {
		models.StatusResolved: {models.RoleAgent, models.RoleAdmin},
	},
	models.StatusResolved: {
		models.StatusClosed: {models.RoleRequester, models.RoleAgent, models.RoleAdmin},
		models.StatusOpen:   {models.RoleRequester},
	},
}

// CanTransition reports whether role may move a ticket from->to.
func CanTransition(from, to, role string) bool {
	if from == to {
		return false
	}
	dests, ok := transitionTable[from]
	if !ok {
		return false
	}
	roles, ok := dests[to]
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// UpdateTicketInput is the validated payload for PATCH /api/tickets/{id}.
type UpdateTicketInput struct {
	Status     string
	Priority   string
	AssigneeID *int64 // nil = leave unchanged; explicit 0 = unassign
}

// Update applies a status/priority/assignee change, enforcing transition rules.
// `actor` is the user making the request.
func (s *TicketService) Update(ctx context.Context, id int64, in UpdateTicketInput, actor *models.User) (*models.Ticket, error) {
	t, err := s.tickets.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("ticketService.Update: %w", err)
	}

	// Status transition check.
	if in.Status != "" && in.Status != t.Status {
		if !CanTransition(t.Status, in.Status, actor.Role) {
			return nil, ErrInvalidTransition
		}
	}

	// Priority change: agent/admin only.
	if in.Priority != "" && in.Priority != t.Priority {
		if actor.Role != models.RoleAgent && actor.Role != models.RoleAdmin {
			return nil, ErrForbidden
		}
	}

	var assignee sql.NullInt64
	if in.AssigneeID != nil {
		// Claiming (assigning to self): agent/admin only.
		if actor.Role != models.RoleAgent && actor.Role != models.RoleAdmin {
			return nil, ErrForbidden
		}
		if *in.AssigneeID == 0 {
			assignee = sql.NullInt64{Valid: false}
		} else {
			assignee = sql.NullInt64{Int64: *in.AssigneeID, Valid: true}
		}
	} else {
		// Leave unchanged: pass the current value through.
		assignee = t.AssigneeID
	}

	status := in.Status
	if status == "" {
		status = t.Status
	}
	priority := in.Priority
	if priority == "" {
		priority = t.Priority
	}

	return s.tickets.Update(ctx, id, status, priority, assignee)
}

// CreateCommentInput is the validated payload for adding a comment.
type CreateCommentInput struct {
	TicketID   int64
	AuthorID   int64
	Body       string
	IsInternal bool
}

// CreateComment adds a comment or internal note. Internal notes are
// agent/admin only.
func (s *TicketService) CreateComment(ctx context.Context, in CreateCommentInput, actor *models.User) (*models.Comment, error) {
	if in.Body == "" {
		return nil, &ValidationError{Fields: validation.FieldErrors{"body": "is required"}}
	}
	if in.IsInternal && actor.Role == models.RoleRequester {
		return nil, ErrForbidden
	}
	c := &models.Comment{
		TicketID:   in.TicketID,
		AuthorID:   in.AuthorID,
		Body:       in.Body,
		IsInternal: in.IsInternal,
	}
	return s.comments.Create(ctx, c)
}
