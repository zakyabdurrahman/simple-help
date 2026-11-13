package validation

import (
	"errors"
	"fmt"
	"regexp"
	"unicode"
)

// emailRE is a pragmatic email pattern, not a full RFC validator. It rejects
// the obviously-bad inputs and accepts the common ones.
var emailRE = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// FieldErrors maps a field name to a human-readable validation message.
type FieldErrors map[string]string

// Error implements error.
func (f FieldErrors) Error() string {
	if len(f) == 0 {
		return "validation failed"
	}
	// stable-ish message; the controller surfaces the map directly.
	var first string
	for k, v := range f {
		first = fmt.Sprintf("%s: %s", k, v)
		break
	}
	return first
}

// IsEmpty reports whether there are no field errors.
func (f FieldErrors) IsEmpty() bool { return len(f) == 0 }

// ErrEmpty signals that no validation errors were collected; useful when
// callers want a sentinel rather than a nil map.
var ErrEmpty = errors.New("no validation errors")

// ValidateRegister checks the public registration payload per spec:
//   - name: 1–100 chars after trim
//   - email: RFC-ish pattern, lowercased by caller before this call
//   - password: 8–72 chars, at least one letter and one digit
//
// Returns the trimmed name, the (still as-given) email, the password, and a
// FieldErrors map. If the map is empty, all fields are valid.
func ValidateRegister(name, email, password string) (string, string, string, FieldErrors) {
	errs := FieldErrors{}

	name = trim(name)
	switch {
	case name == "":
		errs["name"] = "is required"
	case len(name) > 100:
		errs["name"] = "must be 100 characters or fewer"
	}

	email = trim(email)
	if email == "" {
		errs["email"] = "is required"
	} else if !emailRE.MatchString(email) {
		errs["email"] = "is invalid"
	}

	switch {
	case password == "":
		errs["password"] = "is required"
	case len(password) < 8:
		errs["password"] = "must be at least 8 characters"
	case len(password) > 72:
		errs["password"] = "must be 72 characters or fewer"
	case !hasLetter(password) || !hasDigit(password):
		errs["password"] = "must contain at least one letter and one number"
	}

	return name, email, password, errs
}

func trim(s string) string {
	// strip leading/trailing ASCII whitespace; also collapse internal spaces
	// for the name only is overkill here, so just trim ends.
	for len(s) > 0 && isSpace(s[0]) {
		s = s[1:]
	}
	for len(s) > 0 && isSpace(s[len(s)-1]) {
		s = s[:len(s)-1]
	}
	return s
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
