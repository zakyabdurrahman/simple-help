package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------- Register: happy path ----------

func TestRegister_Success(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	w := doJSON(t, h, "POST", "/api/auth/register", map[string]string{
		"name":     "Jane Doe",
		"email":    "jane@example.com",
		"password": "secret123",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)

	for _, key := range []string{"id", "name", "email", "role", "created_at"} {
		if _, ok := body[key]; !ok {
			t.Errorf("response missing %q: %v", key, body)
		}
	}
	if body["name"] != "Jane Doe" {
		t.Errorf("name = %v, want Jane Doe", body["name"])
	}
	if body["email"] != "jane@example.com" {
		t.Errorf("email = %v, want jane@example.com", body["email"])
	}
	if body["role"] != "requester" {
		t.Errorf("role = %v, want requester", body["role"])
	}
	if _, hasHash := body["password_hash"]; hasHash {
		t.Error("password_hash must not be in response")
	}
}

// ---------- Register: invalid JSON ----------

func TestRegister_InvalidJSON(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader([]byte("{not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	body := parseJSON(t, w)
	if body["error"] != "invalid_json" {
		t.Errorf("error = %v, want invalid_json", body["error"])
	}
}

// ---------- Register: validation failures ----------

func TestRegister_ValidationFailures(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	cases := []struct {
		name string
		body map[string]string
	}{
		{"missing name", map[string]string{"email": "a@b.com", "password": "secret123"}},
		{"name too long", map[string]string{"name": string(bytes.Repeat([]byte("x"), 101)), "email": "a@b.com", "password": "secret123"}},
		{"missing email", map[string]string{"name": "A", "password": "secret123"}},
		{"invalid email", map[string]string{"name": "A", "email": "not-an-email", "password": "secret123"}},
		{"password too short", map[string]string{"name": "A", "email": "a@b.com", "password": "ab1"}},
		{"password no letter", map[string]string{"name": "A", "email": "a@b.com", "password": "12345678"}},
		{"password no digit", map[string]string{"name": "A", "email": "a@b.com", "password": "abcdefgh"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := doJSON(t, h, "POST", "/api/auth/register", c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
			body := parseJSON(t, w)
			if body["error"] != "validation_failed" {
				t.Errorf("error = %v, want validation_failed", body["error"])
			}
			if _, ok := body["fields"]; !ok {
				t.Error("expected fields map in response")
			}
		})
	}
}

// ---------- Register: email uniqueness ----------

func TestRegister_DuplicateEmail(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	registerUser(t, h, "Jane", "jane@example.com", "secret123")

	w := doJSON(t, h, "POST", "/api/auth/register", map[string]string{
		"name": "Other", "email": "jane@example.com", "password": "secret123",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if body["error"] != "email_exists" {
		t.Errorf("error = %v, want email_exists", body["error"])
	}
}

// ---------- Register: email case-insensitivity ----------

func TestRegister_EmailCaseInsensitive(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	w := doJSON(t, h, "POST", "/api/auth/register", map[string]string{
		"name": "Jane", "email": "Jane@Example.COM", "password": "secret123",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("first registration failed: %d %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if body["email"] != "jane@example.com" {
		t.Errorf("email should be lowercased to %v, got %v", "jane@example.com", body["email"])
	}

	w = doJSON(t, h, "POST", "/api/auth/register", map[string]string{
		"name": "Other", "email": "jane@example.com", "password": "secret123",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for case variant, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- Healthz ----------

func TestHealthz(t *testing.T) {
	h := newEngine()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("body = %q, want ok", w.Body.String())
	}
}
