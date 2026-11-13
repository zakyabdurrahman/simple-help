package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"simplehelp/src/models"
	"simplehelp/src/repositories"
	"simplehelp/src/validation"
	"golang.org/x/crypto/bcrypt"
)

// AuthService contains the business logic for authentication flows.
// It depends on a UserRepository so it can be unit-tested with a fake.
type AuthService struct {
	users repositories.UserRepository
}

// NewAuthService builds an AuthService backed by the given repository.
func NewAuthService(users repositories.UserRepository) *AuthService {
	return &AuthService{users: users}
}

// RegisterInput is the validated payload for public registration.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// Register creates a new Requester account per spec:
// validate → normalize → uniqueness check → bcrypt → insert.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	name, email, password, errs := validation.ValidateRegister(in.Name, in.Email, in.Password)
	if !errs.IsEmpty() {
		return nil, &ValidationError{Fields: errs}
	}

	// Normalize email to lowercase so case variants collide.
	email = strings.ToLower(email)

	// Uniqueness check.
	if existing, err := s.users.GetByEmail(ctx, email); err == nil && existing != nil {
		return nil, ErrEmailExists
	} else if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("auth.Register: check email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("auth.Register: hash password: %w", err)
	}

	user, err := s.users.Create(ctx, name, email, string(hash), models.RoleRequester)
	if err != nil {
		// A concurrent insert could still race the uniqueness check; the
		// unique constraint will surface a generic error here. Log and bubble
		// up as ErrEmailExists so the client gets a sensible 409.
		slog.Warn("user insert failed", "error", err)
		return nil, fmt.Errorf("auth.Register: create user: %w", err)
	}

	return user, nil
}
