package services

import "errors"

// Sentinel errors that the controller layer maps to HTTP status codes.
var (
	// ErrValidation indicates one or more fields failed validation. The
	// associated Fields map should be attached via ValidationError.
	ErrValidation = errors.New("validation_failed")

	// ErrEmailExists indicates the email is already registered.
	ErrEmailExists = errors.New("email_exists")
)

// ValidationError carries per-field messages alongside ErrValidation.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation failed" }

// Is lets errors.Is(err, ErrValidation) match a *ValidationError.
func (e *ValidationError) Is(target error) bool { return target == ErrValidation }
