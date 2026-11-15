# SimpleHelp  
a helpdesk application to handle customer requests   
## Tech and Architechture  
- the whole app with multiple components will be runnable using docker with docker compose  
- use postgresql as database  
- use Vue 3 composition API for frontend (with vite?)  
- for frontend use Daisy UI component library
- use Go with Gin as backend, migrate for migrations
- use MVC with that separate business logic with db calls to make it testable  
- will be mobile-first (can be fully used on mobile, usable on laptops screen)  
## User Roles  
- Requester (employee) -> Creates tickets, views and comments only on their own tickets, and closes their own tickets.  
- Agent -> Views all tickets, claims (assigns) tickets, changes status and priority, comments, and writes internal notes.  
- Admin -> All agent permissions, plus managing users and roles.  
## Ticket Rules  
| From | To | By |
| --- | --- | --- |
| Open | In Progress | Agent, Admin |
| In Progress | Resolved | Agent, Admin |
| Resolved | Closed | Requester, Agent, Admin |
| Resolved | Open (reopened) | Requester |
| Open | Closed | Requester (canceled), Admin |  
##  Tables  
| Table | Main Columns |
| --- | --- |
| `users` | `id`, `name`, `email` (unique), `password_hash`, `role`, `created_at` |
| `tickets` | `id`, `title`, `description`, `category`, `priority`, `status_id` (status), `requester_id` (users), `assignee_id` (users), `created_at`, `updated_at` |
| `comments` | `id`, `ticket_id` (tickets), `author_id` (users), `body`, `is_internal`, `created_at` |  
| `status` | `id`, `text` |  
## Backend Functions  
| Method | Path | Function | Access |
| --- | --- | --- | --- |
| `POST` | `/api/auth/register` | Register a requester account | Public |
| `POST` | `/api/auth/login` | Log in, returns JWT | Public |
| `GET` | `/api/me` | Current user profile | Logged in |
| `GET` | `/api/tickets` | List tickets (filtering, pagination) | Logged in, role-restricted |
| `POST` | `/api/tickets` | Create a ticket | Logged in |
| `GET` | `/api/tickets/{id}` | Ticket details including comments | Owner, agent, admin |
| `PATCH` | `/api/tickets/{id}` | Update status, priority, assignee | Per transition table |
| `POST` | `/api/tickets/{id}/comments` | Add a comment or internal note | Owner, agent, admin |
| `GET` | `/api/users` | List users | Admin |
| `PATCH` | `/api/users/{id}/role` | Change user role | Admin |

## Frontend Functions  
| Page | Minimum Content |
| :--- | :--- |
| **Login & Register** | Form with client-side validation and server-side error display |
| **Ticket List** | Table with filters, search, pagination, status badges, and priority badges |
| **Create Ticket** | Form with validation, displaying success or failure messages |
| **Ticket Details** | Ticket information, comment history, comment form, and role-based status change buttons |
| **Manage Users (Admin)** | User list and role management |

## Auth

### `POST /api/auth/register`
Create a new user account. Public (no auth required).

Public registration always creates a **Requester** role. Agent/Admin accounts
are created by an Admin via a separate admin endpoint (not covered here).

#### Request body
| Field          | Type   | Required | Rules |
| -------------- | ------ | -------- | ----- |
| `name`         | string | yes      | 1–100 chars, trimmed |
| `email`        | string | yes      | valid email format, lowercased, unique across `users.email` |
| `password`     | string | yes      | 8–72 chars (bcrypt limit), at least 1 letter and 1 number |

#### Validation
- All required fields present and non-empty after trim.
- `email` matched against a basic RFC-ish regex, then lowercased.
- `password` length and complexity checked before hashing.
- On any validation failure → `400 Bad Request` with a JSON body:
  ```json
  { "error": "validation_failed", "fields": { "email": "already in use" } }
  ```

#### Logic
1. Validate input (above).
2. Check `users.email` uniqueness; if exists → `409 Conflict`.
3. Hash `password` with bcrypt (cost 12) → `password_hash`.
4. Insert into `users` with `role = 'requester'`, `created_at = now()`.
5. Return `201 Created` with the public user representation (no `password_hash`).

#### Responses
`201 Created`
```json
{
  "id": 12,
  "name": "Jane Doe",
  "email": "jane@example.com",
  "role": "requester",
  "created_at": "2026-11-13T10:00:00Z"
}
```

`400 Bad Request` — validation failure (see above).
`409 Conflict` — email already registered.
`500 Internal Server Error` — unexpected failure.

#### Security / notes
- `password_hash` is never included in any response.
- Passwords are never logged.
- Email normalization (lowercase + trim) happens before uniqueness check so `Jane@X.com` and `jane@x.com` collide correctly.

