package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"simplehelp/src/models"
)

// UserRepository abstracts user persistence behind a small interface so the
// service layer stays testable without a real database.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, name, email, passwordHash, role string) (*models.User, error)
}

// ErrNotFound is returned when a query expects a row but finds none.
var ErrNotFound = errors.New("not found")

type userRepo struct {
	db *sql.DB
}

// NewUserRepository returns a UserRepository backed by the given *sql.DB.
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `
		SELECT id, name, email, password_hash, role, created_at
		FROM users
		WHERE email = $1
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByEmail: %w", err)
	}
	return u, nil
}

func (r *userRepo) Create(ctx context.Context, name, email, passwordHash, role string) (*models.User, error) {
	const q = `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, email, role, created_at
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, name, email, passwordHash, role).Scan(
		&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("userRepo.Create: %w", err)
	}
	return u, nil
}
