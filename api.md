# SimpleHelp API Specification

HTTP API contract for the SimpleHelp backend.

## Conventions

- **Base URL:** `http://<host>:8080`
- **Content type:** all request and response bodies are `application/json; charset=utf-8`.
- **Timestamps:** RFC 3339 UTC, e.g. `2026-11-13T10:00:00Z`.
- **IDs:** integer (Postgres `BIGSERIAL`).
- **Authentication:** bearer JWT in the `Authorization: <token>` header for protected routes (planned; not yet implemented). Register is public.

## Error envelope

All error responses share the same shape:

```json
{
  "error": "<error_code>",
  "fields": { "<field>": "<message>" }
}
```

`fields` is optional and only present for validation errors. When omitted, the key is absent (not `null`).

### Error codes

| Code               | HTTP | Meaning                                              |
| ------------------ | ---- | ---------------------------------------------------- |
| `invalid_json`     | 400  | Request body was not valid JSON.                     |
| `validation_failed`| 400  | One or more fields failed validation. `fields` lists them. |
| `email_exists`     | 409  | The email is already registered.                     |
| `unauthorized`     | 401  | Missing or invalid auth token. (planned)             |
| `forbidden`        | 403  | Authenticated but not permitted for this action. (planned) |
| `not_found`        | 404  | Resource does not exist. (planned)                   |
| `internal_error`   | 500  | Unexpected server failure.                           |

## Data models

### User (public)

`password_hash` is never included in any response.

```json
{
  "id": 12,
  "name": "Jane Doe",
  "email": "jane@example.com",
  "role": "requester",
  "created_at": "2026-11-13T10:00:00Z"
}
```

Roles (matching the `users.role` CHECK constraint):
- `requester` — creates tickets, views/comments/closes own tickets.
- `agent` — views all tickets, claims/updates tickets, comments, writes internal notes.
- `admin` — all agent permissions plus user/role management.

### Ticket (planned)

```json
{
  "id": 1,
  "title": "string",
  "description": "string",
  "category": "string | null",
  "priority": "low | medium | high",
  "status": "open | in_progress | resolved | closed",
  "requester_id": 12,
  "assignee_id": 42,
  "created_at": "2026-11-13T10:00:00Z",
  "updated_at": "2026-11-13T10:00:00Z"
}
```

### Comment (planned)

```json
{
  "id": 1,
  "ticket_id": 1,
  "author_id": 12,
  "body": "string",
  "is_internal": false,
  "created_at": "2026-11-13T10:00:00Z"
}
```

## Endpoints

### `POST /api/auth/register` ✅ implemented

Create a new user account. Public (no auth). Public registration always creates a **Requester** account; agent/admin accounts are seeded or created by an admin via a separate endpoint (planned).

#### Request body

| Field     | Type   | Required | Rules                                                              |
| --------- | ------ | -------- | ------------------------------------------------------------------ |
| `name`    | string | yes      | 1–100 chars after trim.                                            |
| `email`   | string | yes      | RFC-ish pattern, lowercased before uniqueness check, unique.       |
| `password`| string | yes      | 8–72 chars (bcrypt limit), at least 1 letter and 1 digit.          |

```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "password": "secret123"
}
```

#### Validation

- All required fields present and non-empty after trim.
- `email` matched against `^[^\s@]+@[^\s@]+\.[^\s@]+$`, then lowercased.
- `password` length and complexity checked before hashing.
- Email normalization (trim + lowercase) happens before the uniqueness check so `Jane@X.com` and `jane@x.com` collide correctly.

#### Logic

1. Validate input (above).
2. Check `users.email` uniqueness; if exists → `409 Conflict`.
3. Hash `password` with bcrypt (cost 12) → `password_hash`.
4. Insert into `users` with `role = 'requester'`, `created_at = now()`.
5. Return `201 Created` with the public user representation (no `password_hash`).

#### Responses

**`201 Created`**
```json
{
  "id": 12,
  "name": "Jane Doe",
  "email": "jane@example.com",
  "role": "requester",
  "created_at": "2026-11-13T10:00:00Z"
}
```

**`400 Bad Request`** — malformed JSON.
```json
{ "error": "invalid_json", "fields": { "body": "could not parse JSON body" } }
```

**`400 Bad Request`** — validation failure.
```json
{ "error": "validation_failed", "fields": { "email": "is invalid" } }
```

**`409 Conflict`** — email already registered.
```json
{ "error": "email_exists", "fields": { "email": "already in use" } }
```

**`500 Internal Server Error`** — unexpected failure.
```json
{ "error": "internal_error" }
```

#### Security notes

- `password_hash` is never included in any response.
- Passwords are never logged.
- Email normalization (lowercase + trim) happens before the uniqueness check.

---

### `POST /api/auth/login` ✅ implemented

Log in with email + password. Returns a JWT and the current user.

#### Request body
| Field     | Type   | Required |
| --------- | ------ | -------- |
| `email`   | string | yes      |
| `password`| string | yes      |

```json
{ "email": "jane@example.com", "password": "secret123" }
```

Email is lowercased + trimmed before lookup. Passwords are compared against the stored bcrypt hash.

#### Responses
**`200 OK`**
```json
{
  "token": "eyJhbGciOi...",
  "user": {
    "id": 12,
    "name": "Jane Doe",
    "email": "jane@example.com",
    "role": "requester",
    "created_at": "2026-11-13T10:00:00Z"
  }
}
```

**`400 Bad Request`** — malformed JSON or missing fields.
**`401 Unauthorized`** — wrong password or unknown email.
```json
{ "error": "invalid_credentials" }
```

---

### `GET /api/me` ✅ implemented

Returns the authenticated user's profile. Requires `Authorization: Bearer <token>`.

**`200 OK`** — the public user representation (no `password_hash`).
**`401 Unauthorized`** — missing or invalid token.
```json
{ "error": "unauthorized" }
```

---

## Tickets

All ticket endpoints require authentication (`Authorization: Bearer <token>`).

### `POST /api/tickets` ✅ implemented

Create a ticket. The authenticated user becomes the `requester`. New tickets start with status `open`.

#### Request body
| Field        | Type   | Required | Rules                                            |
| ------------ | ------ | -------- | ------------------------------------------------ |
| `title`      | string | yes      | 1–200 chars.                                     |
| `description`| string | yes      | non-empty.                                       |
| `category`   | string | no       | free text, omitted/null in response if not set.  |
| `priority`   | string | no       | `low` \| `medium` \| `high`; defaults to `medium`.|

#### Responses
**`201 Created`**
```json
{
  "id": 1,
  "title": "My ticket",
  "description": "It broke",
  "category": null,
  "priority": "high",
  "status": "open",
  "requester_id": 12,
  "assignee_id": null,
  "created_at": "2026-11-15T10:00:00Z",
  "updated_at": "2026-11-15T10:00:00Z"
}
```
**`400 Bad Request`** — `invalid_json` or `validation_failed`.

---

### `GET /api/tickets` ✅ implemented

List tickets visible to the caller. Requesters see only their own tickets; agents and admins see all.

#### Query params
| Param    | Default | Notes                                                            |
| -------- | ------- | ---------------------------------------------------------------- |
| `q`      | (none)  | Case-insensitive substring search on title OR description.       |
| `limit`  | 20      | 1–100.                                                           |
| `offset` | 0       | pagination offset.                                               |

#### Response
**`200 OK`**
```json
{ "tickets": [ { ...ticket }, ... ] }
```

---

### `GET /api/tickets/{id}` ✅ implemented

Ticket details including comments. The caller must be the ticket's owner, an agent, or an admin.

**`200 OK`**
```json
{
  "id": 1, "title": "...", ...,
  "comments": [
    { "id": 1, "ticket_id": 1, "author_id": 12, "body": "...", "is_internal": false, "created_at": "..." }
  ]
}
```
**`403 Forbidden`** — caller is not the owner and not agent/admin.
**`404 Not Found`** — ticket does not exist.

---

### `PATCH /api/tickets/{id}` ✅ implemented

Update a ticket's status, priority, and/or assignee. All fields optional; only provided fields change.

#### Request body
| Field        | Type    | Rules                                                                 |
| ------------ | ------- | --------------------------------------------------------------------- |
| `status`     | string  | Must be a legal transition from the current status (see table below). |
| `priority`   | string  | `low` \| `medium` \| `high`. Agent/admin only.                        |
| `assignee_id`| integer | User ID to assign; `0` to unassign. Agent/admin only.                 |

#### Status transitions

| From         | To           | By                            |
| ------------ | ------------ | ----------------------------- |
| Open         | In Progress  | Agent, Admin                  |
| In Progress  | Resolved     | Agent, Admin                  |
| Resolved     | Closed       | Requester, Agent, Admin       |
| Resolved     | Open         | Requester (reopened)          |
| Open         | Closed       | Requester (canceled), Admin   |

#### Responses
**`200 OK`** — updated ticket.
**`403 Forbidden`** — caller attempted a priority/assignee change without agent/admin role.
**`422 Unprocessable Entity`** — illegal status transition.
```json
{ "error": "invalid_transition" }
```
**`404 Not Found`** — ticket does not exist.

---

### `POST /api/tickets/{id}/comments` ✅ implemented

Add a comment or internal note to a ticket. The caller must be permitted to view the ticket (owner, agent, admin).

#### Request body
| Field         | Type    | Required | Rules                                |
| ------------- | ------- | -------- | ------------------------------------ |
| `body`        | string  | yes      | non-empty.                           |
| `is_internal` | boolean | no       | default `false`. Agent/admin only.   |

#### Responses
**`201 Created`**
```json
{
  "id": 5,
  "ticket_id": 1,
  "author_id": 12,
  "body": "...",
  "is_internal": false,
  "created_at": "2026-11-15T10:05:00Z"
}
```
**`403 Forbidden`** — caller cannot view the ticket, or a requester set `is_internal: true`.

---

## User management (admin only)

### `GET /api/users` ✅ implemented

List all users. Admin role required.

#### Query params
| Param    | Default |
| -------- | ------- |
| `limit`  | 50      |
| `offset` | 0       |

#### Response
**`200 OK`**
```json
{ "users": [ { ...user }, ... ] }
```
**`403 Forbidden`** — caller is not an admin.

---

### `PATCH /api/users/{id}/role` ✅ implemented

Change a user's role. Admin only.

#### Request body
| Field  | Type   | Rules                                            |
| ------ | ------ | ------------------------------------------------ |
| `role` | string | `requester` \| `agent` \| `admin`.               |

#### Responses
**`200 OK`** — updated user (no `password_hash`).
**`400 Bad Request`** — `validation_failed` (invalid role value).
**`403 Forbidden`** — caller is not an admin.
**`404 Not Found`** — user does not exist.

---

### `GET /healthz` ✅ implemented

Liveness probe. Returns `200 OK` with body `ok` (plain text, not JSON).
