package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Status constants matching the rows seeded into the `status` table.
const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
)

// Priority constants matching the CHECK constraint on tickets.priority.
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)

// Ticket is the application-level ticket model.
type Ticket struct {
	ID             int64          `json:"id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Category       sql.NullString `json:"-"`
	Priority       string         `json:"priority"`
	Status         string         `json:"status"`
	RequesterID    int64          `json:"requester_id"`
	RequesterName  string         `json:"requester_name"`
	RequesterEmail string         `json:"requester_email"`
	AssigneeID     sql.NullInt64  `json:"-"`
	AssigneeName   sql.NullString `json:"-"`
	AssigneeEmail  sql.NullString `json:"-"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// MarshalJSON produces the public JSON representation of a ticket, exposing
// category/assignee as nullable strings/integers rather than sql.Null*.
func (t Ticket) MarshalJSON() ([]byte, error) {
	type out struct {
		ID             int64  `json:"id"`
		Title          string `json:"title"`
		Description    string `json:"description"`
		Category       any    `json:"category"`
		Priority       string `json:"priority"`
		Status         string `json:"status"`
		RequesterID    int64  `json:"requester_id"`
		RequesterName  string `json:"requester_name"`
		RequesterEmail string `json:"requester_email"`
		AssigneeID     any    `json:"assignee_id"`
		AssigneeName   any    `json:"assignee_name"`
		AssigneeEmail  any    `json:"assignee_email"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	}
	o := out{
		ID: t.ID, Title: t.Title, Description: t.Description,
		Priority: t.Priority, Status: t.Status,
		RequesterID: t.RequesterID, RequesterName: t.RequesterName, RequesterEmail: t.RequesterEmail,
		CreatedAt: t.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339Nano),
	}
	if t.Category.Valid {
		o.Category = t.Category.String
	}
	if t.AssigneeID.Valid {
		o.AssigneeID = t.AssigneeID.Int64
	}
	if t.AssigneeName.Valid {
		o.AssigneeName = t.AssigneeName.String
	}
	if t.AssigneeEmail.Valid {
		o.AssigneeEmail = t.AssigneeEmail.String
	}
	return json.Marshal(o)
}

// Comment is the application-level comment model.
type Comment struct {
	ID          int64     `json:"id"`
	TicketID    int64     `json:"ticket_id"`
	AuthorID    int64     `json:"author_id"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	Body        string    `json:"body"`
	IsInternal  bool      `json:"is_internal"`
	CreatedAt   time.Time `json:"created_at"`
}
