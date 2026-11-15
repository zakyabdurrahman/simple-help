package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"simplehelp/src/auth"
	"simplehelp/src/models"
	"simplehelp/src/repositories"
	"simplehelp/src/validation"
)

// AuthService contains the business logic for authentication flows.
// It depends on a UserRepository so it can be unit-tested with a fake.
type AuthService struct {
	users     repositories.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

// NewAuthService builds an AuthService backed by the given repository.
// jwtSecret and jwtExpiry configure the tokens issued by Login.
func NewAuthService(users repositories.UserRepository, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
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

// LoginInput is the payload for POST /api/auth/login.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult is returned by a successful login.
type LoginResult struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// Login verifies credentials and returns a signed JWT.
func (s *AuthService) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || in.Password == "" {
		return nil, &ValidationError{Fields: validation.FieldErrors{
			"email":    "is required",
			"password": "is required",
		}}
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("auth.Login: lookup: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Token is generated here (not in the controller) so the controller stays thin.
	token, err := auth.Generate(s.jwtSecret, user.ID, user.Role, s.jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("auth.Login: token: %w", err)
	}

	return &LoginResult{Token: token, User: user}, nil
}

// Me returns the current user by ID.
func (s *AuthService) Me(ctx context.Context, id int64) (*models.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("auth.Me: %w", err)
	}
	return user, nil
}
