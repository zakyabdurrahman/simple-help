package services

import (
	"context"
	"errors"
	"fmt"

	"simplehelp/src/models"
	"simplehelp/src/repositories"
	"simplehelp/src/validation"
)

// UserService contains admin user-management logic.
type UserService struct {
	users repositories.UserRepository
}

// NewUserService builds a UserService.
func NewUserService(users repositories.UserRepository) *UserService {
	return &UserService{users: users}
}

// List returns a page of users.
func (s *UserService) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.users.List(ctx, limit, offset)
}

// UpdateRole changes a user's role. Validates the role value.
func (s *UserService) UpdateRole(ctx context.Context, id int64, role string) (*models.User, error) {
	if role != models.RoleRequester && role != models.RoleAgent && role != models.RoleAdmin {
		return nil, &ValidationError{Fields: validation.FieldErrors{
			"role": "must be one of requester, agent, admin",
		}}
	}
	user, err := s.users.UpdateRole(ctx, id, role)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("userService.UpdateRole: %w", err)
	}
	return user, nil
}
