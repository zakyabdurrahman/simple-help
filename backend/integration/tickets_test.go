package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"simplehelp/src/models"
)

// ---------- Ticket: create ----------

func TestTicket_Create_Requester(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	token, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "POST", "/api/tickets", token, map[string]string{
		"title": "My ticket", "description": "It broke", "priority": "high",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if body["title"] != "My ticket" {
		t.Errorf("title = %v", body["title"])
	}
	if body["status"] != models.StatusOpen {
		t.Errorf("status = %v, want open", body["status"])
	}
	if body["priority"] != "high" {
		t.Errorf("priority = %v, want high", body["priority"])
	}
	if body["category"] != nil {
		t.Errorf("category should be null, got %v", body["category"])
	}
}

func TestTicket_Create_DefaultPriority(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	token, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "POST", "/api/tickets", token, map[string]string{
		"title": "T", "description": "D",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["priority"] != "medium" {
		t.Error("default priority should be medium")
	}
}

func TestTicket_Create_ValidationFailures(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	token, _ := seedRoleToken(t, h, models.RoleRequester)

	cases := []struct {
		name string
		body map[string]string
	}{
		{"missing title", map[string]string{"description": "d"}},
		{"missing description", map[string]string{"title": "t"}},
		{"bad priority", map[string]string{"title": "t", "description": "d", "priority": "urgent"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := doJSONAuth(t, h, "POST", "/api/tickets", token, c.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
			if parseJSON(t, w)["error"] != "validation_failed" {
				t.Error("expected validation_failed")
			}
		})
	}
}

func TestTicket_Create_NoAuth(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	w := doJSONAuth(t, h, "POST", "/api/tickets", "", map[string]string{"title": "t", "description": "d"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ---------- Ticket: list ----------

func TestTicket_List_RequesterSeesOwn(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, rEmail := seedRoleToken(t, h, models.RoleRequester)
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	// Requester creates a ticket.
	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "r1", "description": "d"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}
	// Agent creates a ticket (requester=agent).
	w2 := doJSONAuth(t, h, "POST", "/api/tickets", aTok, map[string]string{"title": "a1", "description": "d"})
	if w2.Code != http.StatusCreated {
		t.Fatalf("agent create failed: %d %s", w2.Code, w2.Body.String())
	}

	// Requester lists -> sees only their own (1).
	w = doJSONAuth(t, h, "GET", "/api/tickets", rTok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list failed: %d %s", w.Code, w.Body.String())
	}
	tickets := parseJSON(t, w)["tickets"].([]any)
	if len(tickets) != 1 {
		t.Errorf("requester should see 1 ticket, got %d", len(tickets))
	}
	_ = rEmail

	// Agent lists -> sees all (2).
	w = doJSONAuth(t, h, "GET", "/api/tickets", aTok, nil)
	tickets = parseJSON(t, w)["tickets"].([]any)
	if len(tickets) != 2 {
		t.Errorf("agent should see 2 tickets, got %d", len(tickets))
	}
}

// ---------- Ticket: get ----------

func TestTicket_Get_Owner(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "GET", "/api/tickets/"+itoaStr(id), rTok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if _, ok := body["comments"]; !ok {
		t.Error("expected comments array in detail")
	}
}

func TestTicket_Get_NotOwner_Forbidden(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	r1Tok, _ := seedRoleToken(t, h, models.RoleRequester)
	r2Tok, _ := seedRoleToken(t, h, models.RoleRequester) // different email

	// r1 creates a ticket.
	w := doJSONAuth(t, h, "POST", "/api/tickets", r1Tok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	// r2 tries to view -> 403.
	w = doJSONAuth(t, h, "GET", "/api/tickets/"+itoaStr(id), r2Tok, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTicket_Get_NotFound(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "GET", "/api/tickets/999999", rTok, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTicket_Get_AgentCanSeeAny(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "GET", "/api/tickets/"+itoaStr(id), aTok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("agent should see any ticket, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- Ticket: transitions ----------

func TestTicket_Transition_AgentToInProgress(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "PATCH", "/api/tickets/"+itoaStr(id), aTok, map[string]string{"status": "in_progress"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["status"] != "in_progress" {
		t.Error("status should be in_progress")
	}
}

func TestTicket_Transition_Illegal(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	// Open -> Resolved is not allowed (must go via in_progress).
	w = doJSONAuth(t, h, "PATCH", "/api/tickets/"+itoaStr(id), aTok, map[string]string{"status": "resolved"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["error"] != "invalid_transition" {
		t.Error("expected invalid_transition")
	}
}

func TestTicket_Transition_RequesterCloseOwn(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	// Requester cancels (open -> closed).
	w = doJSONAuth(t, h, "PATCH", "/api/tickets/"+itoaStr(id), rTok, map[string]string{"status": "closed"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["status"] != "closed" {
		t.Error("status should be closed")
	}
}

func TestTicket_PriorityChange_RequesterForbidden(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "PATCH", "/api/tickets/"+itoaStr(id), rTok, map[string]string{"priority": "high"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTicket_Assign_Agent(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)
	aTok, aEmail := seedRoleToken(t, h, models.RoleAgent)
	agentID := userIDByEmail(t, aEmail)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "PATCH", "/api/tickets/"+itoaStr(id), aTok, map[string]any{"assignee_id": agentID})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["assignee_id"] != float64(agentID) {
		t.Errorf("assignee_id = %v, want %d", parseJSON(t, w)["assignee_id"], agentID)
	}
}

// ---------- Ticket: comments ----------

func TestComment_RequesterOnOwnTicket(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "POST", "/api/tickets/"+itoaStr(id)+"/comments", rTok, map[string]string{"body": "hello"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if body["body"] != "hello" {
		t.Errorf("body = %v", body["body"])
	}
	if body["is_internal"] != false {
		t.Error("should not be internal by default")
	}
}

func TestComment_Internal_RequesterForbidden(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "POST", "/api/tickets/"+itoaStr(id)+"/comments", rTok, map[string]any{"body": "note", "is_internal": true})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestComment_Internal_AgentAllowed(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	w := doJSONAuth(t, h, "POST", "/api/tickets", rTok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "POST", "/api/tickets/"+itoaStr(id)+"/comments", aTok, map[string]any{"body": "note", "is_internal": true})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["is_internal"] != true {
		t.Error("should be internal")
	}
}

func TestComment_NotOwner_Forbidden(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	r1Tok, _ := seedRoleToken(t, h, models.RoleRequester)
	r2Tok, _ := seedRoleToken(t, h, models.RoleRequester)

	w := doJSONAuth(t, h, "POST", "/api/tickets", r1Tok, map[string]string{"title": "t", "description": "d"})
	id := int64(parseJSON(t, w)["id"].(float64))

	w = doJSONAuth(t, h, "POST", "/api/tickets/"+itoaStr(id)+"/comments", r2Tok, map[string]string{"body": "hi"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- User admin endpoints ----------

func TestUsers_List_AdminOnly(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	aTok, _ := seedRoleToken(t, h, models.RoleAdmin)
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)

	// Admin can list.
	w := doJSONAuth(t, h, "GET", "/api/users", aTok, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("admin list failed: %d %s", w.Code, w.Body.String())
	}
	users := parseJSON(t, w)["users"].([]any)
	if len(users) < 2 {
		t.Errorf("expected >=2 users, got %d", len(users))
	}

	// Requester cannot list -> 403.
	w = doJSONAuth(t, h, "GET", "/api/users", rTok, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUsers_UpdateRole_Admin(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	aTok, _ := seedRoleToken(t, h, models.RoleAdmin)
	_, rEmail := seedRoleToken(t, h, models.RoleRequester)
	rid := userIDByEmail(t, rEmail)

	w := doJSONAuth(t, h, "PATCH", "/api/users/"+itoaStr(rid)+"/role", aTok, map[string]string{"role": "agent"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["role"] != "agent" {
		t.Errorf("role = %v, want agent", parseJSON(t, w)["role"])
	}
}

func TestUsers_UpdateRole_InvalidRole(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	aTok, _ := seedRoleToken(t, h, models.RoleAdmin)
	_, rEmail := seedRoleToken(t, h, models.RoleRequester)
	rid := userIDByEmail(t, rEmail)

	w := doJSONAuth(t, h, "PATCH", "/api/users/"+itoaStr(rid)+"/role", aTok, map[string]string{"role": "superadmin"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if parseJSON(t, w)["error"] != "validation_failed" {
		t.Error("expected validation_failed")
	}
}

func TestUsers_UpdateRole_NonAdminForbidden(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	rTok, _ := seedRoleToken(t, h, models.RoleRequester)
	_, aEmail := seedRoleToken(t, h, models.RoleAgent)
	aid := userIDByEmail(t, aEmail)

	w := doJSONAuth(t, h, "PATCH", "/api/users/"+itoaStr(aid)+"/role", rTok, map[string]string{"role": "admin"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------- Ticket: search ----------

func TestTicket_Search_TitleMatch(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	// Create three tickets with distinct titles/descriptions.
	doJSONAuth(t, h, "POST", "/api/tickets", aTok, map[string]string{
		"title": "Printer not working", "description": "It stopped",
	})
	doJSONAuth(t, h, "POST", "/api/tickets", aTok, map[string]string{
		"title": "Login issue", "description": "Cannot login",
	})
	doJSONAuth(t, h, "POST", "/api/tickets", aTok, map[string]string{
		"title": "Email broken", "description": "printer queue jammed",
	})

	// Search "printer" -> matches title of #1 and description of #3.
	req := httptest.NewRequest("GET", "/api/tickets?q=printer", nil)
	req.Header.Set("Authorization", "Bearer "+aTok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	tickets := parseJSON(t, w)["tickets"].([]any)
	if len(tickets) != 2 {
		t.Errorf("expected 2 matching tickets, got %d", len(tickets))
	}
}

func TestTicket_Search_CaseInsensitive(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	doJSONAuth(t, h, "POST", "/api/tickets", aTok, map[string]string{
		"title": "Printer Not Working", "description": "It stopped",
	})

	req := httptest.NewRequest("GET", "/api/tickets?q=PRINTER", nil)
	req.Header.Set("Authorization", "Bearer "+aTok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	tickets := parseJSON(t, w)["tickets"].([]any)
	if len(tickets) != 1 {
		t.Errorf("case-insensitive match should find 1, got %d", len(tickets))
	}
}

func TestTicket_Search_NoMatch(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	doJSONAuth(t, h, "POST", "/api/tickets", aTok, map[string]string{
		"title": "Printer not working", "description": "It stopped",
	})

	req := httptest.NewRequest("GET", "/api/tickets?q=nonexistentterm", nil)
	req.Header.Set("Authorization", "Bearer "+aTok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	tickets := parseJSON(t, w)["tickets"].([]any)
	if len(tickets) != 0 {
		t.Errorf("expected 0 matches, got %d", len(tickets))
	}
}

func TestTicket_Search_RequesterScoped(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	r1Tok, _ := seedRoleToken(t, h, models.RoleRequester)
	r2Tok, _ := seedRoleToken(t, h, models.RoleRequester)

	// r1 creates a ticket matching "printer".
	doJSONAuth(t, h, "POST", "/api/tickets", r1Tok, map[string]string{
		"title": "Printer broken", "description": "help",
	})
	// r2 creates a ticket also matching "printer" — but r1 should not see it.
	doJSONAuth(t, h, "POST", "/api/tickets", r2Tok, map[string]string{
		"title": "Printer jam", "description": "help",
	})

	req := httptest.NewRequest("GET", "/api/tickets?q=printer", nil)
	req.Header.Set("Authorization", "Bearer "+r1Tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	tickets := parseJSON(t, w)["tickets"].([]any)
	if len(tickets) != 1 {
		t.Errorf("requester should only see their own matching ticket (1), got %d", len(tickets))
	}
}

func TestTicket_Search_EmptyReturnsAll(t *testing.T) {
	resetUsers(t)
	h := newEngine()
	aTok, _ := seedRoleToken(t, h, models.RoleAgent)

	doJSONAuth(t, h, "POST", "/api/tickets", aTok, map[string]string{"title": "A", "description": "x"})
	doJSONAuth(t, h, "POST", "/api/tickets", aTok, map[string]string{"title": "B", "description": "y"})

	// No q param -> all tickets returned.
	req := httptest.NewRequest("GET", "/api/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+aTok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	tickets := parseJSON(t, w)["tickets"].([]any)
	if len(tickets) != 2 {
		t.Errorf("without q, expected all 2 tickets, got %d", len(tickets))
	}
}

// itoaStr is a tiny int64->string helper to avoid importing strconv in tests.
func itoaStr(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
