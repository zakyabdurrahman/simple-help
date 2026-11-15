// Type definitions matching the backend API models.

export type Role = 'requester' | 'agent' | 'admin'

export interface User {
  id: number
  name: string
  email: string
  role: Role
  created_at: string
}

export type TicketStatus = 'open' | 'in_progress' | 'resolved' | 'closed'
export type Priority = 'low' | 'medium' | 'high'

export interface Ticket {
  id: number
  title: string
  description: string
  category: string | null
  priority: Priority
  status: TicketStatus
  requester_id: number
  assignee_id: number | null
  created_at: string
  updated_at: string
}

export interface Comment {
  id: number
  ticket_id: number
  author_id: number
  body: string
  is_internal: boolean
  created_at: string
}

export interface TicketDetail extends Ticket {
  comments: Comment[]
}

export interface LoginResponse {
  token: string
  user: User
}

// Error envelope from the API: { error: "code", fields?: { field: msg } }
export interface ApiError {
  error: string
  fields?: Record<string, string>
}

export interface ListResponse<T> {
  tickets?: T[]
  users?: T[]
}
