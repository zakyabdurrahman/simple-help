package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"simplehelp/src/models"
)

// TicketRepository abstracts ticket persistence.
type TicketRepository interface {
	Create(ctx context.Context, t *models.Ticket) (*models.Ticket, error)
	GetByID(ctx context.Context, id int64) (*models.Ticket, error)
	List(ctx context.Context, requesterID int64, role string, limit, offset int) ([]*models.Ticket, error)
	Update(ctx context.Context, id int64, status, priority string, assigneeID sql.NullInt64) (*models.Ticket, error)
}

// CommentRepository abstracts comment persistence.
type CommentRepository interface {
	Create(ctx context.Context, c *models.Comment) (*models.Comment, error)
	ListByTicket(ctx context.Context, ticketID int64) ([]*models.Comment, error)
}

type ticketRepo struct {
	db *sql.DB
}

// NewTicketRepository returns a TicketRepository backed by the given *sql.DB.
func NewTicketRepository(db *sql.DB) TicketRepository {
	return &ticketRepo{db: db}
}

func (r *ticketRepo) Create(ctx context.Context, t *models.Ticket) (*models.Ticket, error) {
	const q = `
		INSERT INTO tickets (title, description, category, priority, status_id, requester_id)
		VALUES ($1, $2, $3, $4, (SELECT id FROM status WHERE text = $5), $6)
		RETURNING id, title, description, category, priority,
		          (SELECT text FROM status WHERE id = tickets.status_id) AS status,
		          requester_id, assignee_id, created_at, updated_at
	`
	var cat sql.NullString
	if t.Category.Valid && t.Category.String != "" {
		cat = t.Category
	}
	row := r.db.QueryRowContext(ctx, q,
		t.Title, t.Description, cat, t.Priority, models.StatusOpen, t.RequesterID,
	)
	out := &models.Ticket{}
	if err := scanTicket(row, out); err != nil {
		return nil, fmt.Errorf("ticketRepo.Create: %w", err)
	}
	return out, nil
}

func (r *ticketRepo) GetByID(ctx context.Context, id int64) (*models.Ticket, error) {
	const q = `
		SELECT t.id, t.title, t.description, t.category, t.priority,
		       s.text AS status, t.requester_id, t.assignee_id, t.created_at, t.updated_at
		FROM tickets t
		JOIN status s ON s.id = t.status_id
		WHERE t.id = $1
	`
	row := r.db.QueryRowContext(ctx, q, id)
	out := &models.Ticket{}
	if err := scanTicket(row, out); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("ticketRepo.GetByID: %w", err)
	}
	return out, nil
}

func (r *ticketRepo) List(ctx context.Context, requesterID int64, role string, limit, offset int) ([]*models.Ticket, error) {
	// Requesters only see their own tickets; agents/admins see all.
	q := `
		SELECT t.id, t.title, t.description, t.category, t.priority,
		       s.text AS status, t.requester_id, t.assignee_id, t.created_at, t.updated_at
		FROM tickets t
		JOIN status s ON s.id = t.status_id
	`
	args := []any{}
	if role == models.RoleRequester {
		q += " WHERE t.requester_id = $1"
		args = append(args, requesterID)
	}
	q += " ORDER BY t.id DESC LIMIT $" + itoa(len(args)+1) + " OFFSET $" + itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ticketRepo.List: %w", err)
	}
	defer rows.Close()

	var out []*models.Ticket
	for rows.Next() {
		t := &models.Ticket{}
		if err := scanTicket(rows, t); err != nil {
			return nil, fmt.Errorf("ticketRepo.List scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *ticketRepo) Update(ctx context.Context, id int64, status, priority string, assigneeID sql.NullInt64) (*models.Ticket, error) {
	const q = `
		UPDATE tickets SET
		    status_id  = (SELECT id FROM status WHERE text = $2),
		    priority   = COALESCE(NULLIF($3, ''), priority),
		    assignee_id = $4
		WHERE id = $1
		RETURNING id, title, description, category, priority,
		          (SELECT text FROM status WHERE id = tickets.status_id) AS status,
		          requester_id, assignee_id, created_at, updated_at
	`
	row := r.db.QueryRowContext(ctx, q, id, status, priority, assigneeID)
	out := &models.Ticket{}
	if err := scanTicket(row, out); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("ticketRepo.Update: %w", err)
	}
	return out, nil
}

// scanner matches both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanTicket(s scanner, t *models.Ticket) error {
	return s.Scan(
		&t.ID, &t.Title, &t.Description, &t.Category, &t.Priority,
		&t.Status, &t.RequesterID, &t.AssigneeID, &t.CreatedAt, &t.UpdatedAt,
	)
}

func itoa(n int) string {
	// tiny helper to avoid strconv import for building $N placeholders
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// ---- comment repo ----

type commentRepo struct {
	db *sql.DB
}

// NewCommentRepository returns a CommentRepository backed by the given *sql.DB.
func NewCommentRepository(db *sql.DB) CommentRepository {
	return &commentRepo{db: db}
}

func (r *commentRepo) Create(ctx context.Context, c *models.Comment) (*models.Comment, error) {
	const q = `
		INSERT INTO comments (ticket_id, author_id, body, is_internal)
		VALUES ($1, $2, $3, $4)
		RETURNING id, ticket_id, author_id, body, is_internal, created_at
	`
	out := &models.Comment{}
	err := r.db.QueryRowContext(ctx, q, c.TicketID, c.AuthorID, c.Body, c.IsInternal).Scan(
		&out.ID, &out.TicketID, &out.AuthorID, &out.Body, &out.IsInternal, &out.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("commentRepo.Create: %w", err)
	}
	return out, nil
}

func (r *commentRepo) ListByTicket(ctx context.Context, ticketID int64) ([]*models.Comment, error) {
	const q = `
		SELECT id, ticket_id, author_id, body, is_internal, created_at
		FROM comments
		WHERE ticket_id = $1
		ORDER BY id
	`
	rows, err := r.db.QueryContext(ctx, q, ticketID)
	if err != nil {
		return nil, fmt.Errorf("commentRepo.ListByTicket: %w", err)
	}
	defer rows.Close()

	var out []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.IsInternal, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("commentRepo.ListByTicket scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
