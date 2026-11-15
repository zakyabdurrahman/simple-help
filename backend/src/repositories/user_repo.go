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
	GetByID(ctx context.Context, id int64) (*models.User, error)
	Create(ctx context.Context, name, email, passwordHash, role string) (*models.User, error)
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	UpdateRole(ctx context.Context, id int64, role string) (*models.User, error)
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

func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	const q = `
		SELECT id, name, email, password_hash, role, created_at
		FROM users
		WHERE id = $1
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByID: %w", err)
	}
	return u, nil
}

func (r *userRepo) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	const q = `
		SELECT id, name, email, role, created_at
		FROM users
		ORDER BY id
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("userRepo.List: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("userRepo.List scan: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *userRepo) UpdateRole(ctx context.Context, id int64, role string) (*models.User, error) {
	const q = `
		UPDATE users SET role = $2
		WHERE id = $1
		RETURNING id, name, email, role, created_at
	`
	u := &models.User{}
	err := r.db.QueryRowContext(ctx, q, id, role).Scan(
		&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.UpdateRole: %w", err)
	}
	return u, nil
}
