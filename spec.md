# SimpleHelp  
a helpdesk application to handle customer requests   
## Tech and Architechture  
- the whole app with multiple components will be runnable using docker with docker compose  
- use postgresql as database  
- use Vue 3 composition API for frontend (with vite?)  
- use Go with Gin as backend  
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
| `tickets` | `id`, `title`, `description`, `category`, `priority`, `status`, `requester_id` (users), `assignee_id` (users), `created_at`, `updated_at` |
| `comments` | `id`, `ticket_id` (tickets), `author_id` (users), `body`, `is_internal`, `created_at` |  
## Business Flow  

