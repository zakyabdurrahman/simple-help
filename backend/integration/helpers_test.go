// Package integration contains end-to-end integration tests for the
// SimpleHelp API. Tests run against a real PostgreSQL database (the `db`
// service from docker-compose) and exercise the full HTTP stack via
// httptest.
//
// Run inside the container:
//
//	docker compose --profile test run --rm test
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"simplehelp/src/config"
	"simplehelp/src/db"
	"simplehelp/src/server"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Shared test config + DB connection. The JWT secret/expiry are fixed so
// tokens are deterministic and parseable in tests.
const (
	testJWTSecret = "test-secret"
	testJWTExpiry = time.Hour
)

var (
	testDB *sql.DB
	testCfg config.Config
)

func TestMain(m *testing.M) {
	cfg, err := config.Load()
	if err != nil {
		panic("config load: " + err.Error())
	}
	testCfg = cfg
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := db.Connect(ctx, cfg.DSN(), 30*time.Second)
	if err != nil {
		panic("db connect: " + err.Error())
	}
	testDB = database
	defer testDB.Close()
	m.Run()
}

// resetUsers truncates the users table (and dependent tickets/comments) so
// each test starts clean. Order matters: comments → tickets → users, to
// satisfy foreign keys. TRUNCATE ... CASCADE would also work but explicit
// ordering is clearer.
func resetUsers(t *testing.T) {
	t.Helper()
	if _, err := testDB.Exec(`DELETE FROM comments`); err != nil {
		t.Fatalf("delete comments: %v", err)
	}
	if _, err := testDB.Exec(`DELETE FROM tickets`); err != nil {
		t.Fatalf("delete tickets: %v", err)
	}
	if _, err := testDB.Exec(`DELETE FROM users`); err != nil {
		t.Fatalf("delete users: %v", err)
	}
}

// newEngine builds a fresh Gin engine wired to the shared test DB.
func newEngine() http.Handler {
	return server.New(testDB, testJWTSecret, testJWTExpiry)
}

// doJSON sends a request with the given JSON body and returns the response.
func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// doJSONAuth is doJSON with a bearer token.
func doJSONAuth(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("parse response body %q: %v", w.Body.String(), err)
	}
	return m
}

// registerUser creates a user via the register endpoint and returns the
// response body. Fatals on non-201.
func registerUser(t *testing.T, h http.Handler, name, email, password string) map[string]any {
	t.Helper()
	w := doJSON(t, h, "POST", "/api/auth/register", map[string]string{
		"name": name, "email": email, "password": password,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("register %s failed: %d %s", email, w.Code, w.Body.String())
	}
	return parseJSON(t, w)
}

// loginUser logs in and returns the response body {token, user}. Fatals on
// non-200.
func loginUser(t *testing.T, h http.Handler, email, password string) map[string]any {
	t.Helper()
	w := doJSON(t, h, "POST", "/api/auth/login", map[string]string{
		"email": email, "password": password,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login %s failed: %d %s", email, w.Code, w.Body.String())
	}
	return parseJSON(t, w)
}

// loginToken logs in and returns just the JWT string.
func loginToken(t *testing.T, h http.Handler, email, password string) string {
	t.Helper()
	return loginUser(t, h, email, password)["token"].(string)
}
