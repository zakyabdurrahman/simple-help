package services

import "errors"

// Sentinel errors that the controller layer maps to HTTP status codes.
var (
	// ErrValidation indicates one or more fields failed validation. The
	// associated Fields map should be attached via ValidationError.
	ErrValidation = errors.New("validation_failed")

	// ErrEmailExists indicates the email is already registered.
	ErrEmailExists = errors.New("email_exists")

	// ErrInvalidCredentials indicates email/password did not match.
	ErrInvalidCredentials = errors.New("invalid_credentials")

	// ErrNotFound indicates the requested resource was not found.
	ErrNotFound = errors.New("not_found")

	// ErrInvalidTransition indicates an illegal ticket status transition.
	ErrInvalidTransition = errors.New("invalid_transition")

	// ErrForbidden indicates an authenticated user lacks permission.
	ErrForbidden = errors.New("forbidden")
)

// ValidationError carries per-field messages alongside ErrValidation.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation failed" }

// Is lets errors.Is(err, ErrValidation) match a *ValidationError.
func (e *ValidationError) Is(target error) bool { return target == ErrValidation }
