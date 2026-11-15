package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------- Login: happy path ----------

func TestLogin_Success(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	registerUser(t, h, "Jane", "jane@example.com", "secret123")

	body := loginUser(t, h, "jane@example.com", "secret123")

	if _, ok := body["token"]; !ok {
		t.Fatalf("response missing token: %v", body)
	}
	token, ok := body["token"].(string)
	if !ok || token == "" {
		t.Fatalf("token not a non-empty string: %v", body["token"])
	}
	user, ok := body["user"].(map[string]any)
	if !ok {
		t.Fatalf("user not an object: %v", body["user"])
	}
	if user["email"] != "jane@example.com" {
		t.Errorf("user.email = %v, want jane@example.com", user["email"])
	}
	if user["role"] != "requester" {
		t.Errorf("user.role = %v, want requester", user["role"])
	}
	if _, hasHash := user["password_hash"]; hasHash {
		t.Error("password_hash must not be in login response")
	}
}

// ---------- Login: case-insensitive email ----------

func TestLogin_EmailCaseInsensitive(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	registerUser(t, h, "Jane", "jane@example.com", "secret123")

	// Mixed-case email should still log in.
	body := loginUser(t, h, "Jane@Example.COM", "secret123")
	if body["token"] == nil {
		t.Fatal("expected token for case-variant email")
	}
}

// ---------- Login: wrong password ----------

func TestLogin_WrongPassword(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	registerUser(t, h, "Jane", "jane@example.com", "secret123")

	w := doJSON(t, h, "POST", "/api/auth/login", map[string]string{
		"email": "jane@example.com", "password": "wrongpass1",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if body["error"] != "invalid_credentials" {
		t.Errorf("error = %v, want invalid_credentials", body["error"])
	}
}

// ---------- Login: unknown user ----------

func TestLogin_UnknownUser(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	w := doJSON(t, h, "POST", "/api/auth/login", map[string]string{
		"email": "nobody@example.com", "password": "secret123",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- Login: invalid JSON ----------

func TestLogin_InvalidJSON(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader([]byte("{bad json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if parseJSON(t, w)["error"] != "invalid_json" {
		t.Error("expected invalid_json")
	}
}

// ---------- Me: happy path ----------

func TestMe_Success(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	registerUser(t, h, "Jane", "jane@example.com", "secret123")
	token := loginToken(t, h, "jane@example.com", "secret123")

	w := doJSONAuth(t, h, "GET", "/api/me", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if body["email"] != "jane@example.com" {
		t.Errorf("email = %v, want jane@example.com", body["email"])
	}
	if body["role"] != "requester" {
		t.Errorf("role = %v, want requester", body["role"])
	}
	if _, hasHash := body["password_hash"]; hasHash {
		t.Error("password_hash must not be in /api/me response")
	}
}

// ---------- Me: no auth header ----------

func TestMe_NoAuth(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	w := doJSONAuth(t, h, "GET", "/api/me", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["error"] != "unauthorized" {
		t.Error("expected unauthorized")
	}
}

// ---------- Me: invalid token ----------

func TestMe_InvalidToken(t *testing.T) {
	resetUsers(t)
	h := newEngine()

	w := doJSONAuth(t, h, "GET", "/api/me", "garbage.token.here", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
