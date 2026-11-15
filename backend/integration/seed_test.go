package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"simplehelp/src/models"
)

// seedUser inserts a user with the given role directly into the DB and
// returns the created user. Password is always "password123" (hashed with
// bcrypt, matching the auth service). Use this to set up agent/admin users
// that can't be created via the public register endpoint.
func seedUser(t *testing.T, name, email, role string) *models.User {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	const q = `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, email, role, created_at
	`
	u := &models.User{}
	err = testDB.QueryRowContext(ctx, q, name, email, string(hash), role).Scan(
		&u.ID, &u.Name, &u.Email, &u.Role, &u.CreatedAt,
	)
	if err != nil {
		t.Fatalf("seedUser %s: %v", email, err)
	}
	return u
}

// seedCounter makes each seeded user's email unique within a test run.
var seedCounter int

// seedRoleToken ensures a user with the given role exists (register for
// requester, DB-seed for agent/admin) and returns a login token plus the
// user's email. All accounts use password "password123". Each call produces
// a distinct email so multiple users of the same role can be seeded in one
// test.
func seedRoleToken(t *testing.T, h http.Handler, role string) (token, email string) {
	t.Helper()
	seedCounter++
	email = fmt.Sprintf("%s%d@example.com", role, seedCounter)
	name := fmt.Sprintf("%s user %d", role, seedCounter)
	password := "password123"
	switch role {
	case models.RoleRequester:
		registerUser(t, h, name, email, password)
	default:
		seedUser(t, name, email, role)
	}
	token = loginToken(t, h, email, password)
	return token, email
}

// userIDByEmail looks up a user's ID from the DB by email.
func userIDByEmail(t *testing.T, email string) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id int64
	err := testDB.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err != nil {
		t.Fatalf("userIDByEmail %s: %v", email, err)
	}
	return id
}
