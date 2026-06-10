// Cliente HTTP da API Obsidian Chamados. Em dev, a base é vazia e o proxy do
// Vite encaminha para o backend Go; em produção, defina VITE_API_URL.

const BASE = import.meta.env.VITE_API_URL ?? ''

export interface User {
  id: number
  name: string
  email: string
  role: Role
}

export interface Tenant {
  id: number
  name: string
  slug: string
}

export interface AuthResult {
  access_token: string
  refresh_token: string
  user: User
  tenant: Tenant
}

export interface RegisterInput {
  tenant_name: string
  slug: string
  name: string
  email: string
  password: string
}

export interface LoginInput {
  slug: string
  email: string
  password: string
}

export type TicketStatus = 'open' | 'in_progress' | 'waiting_customer' | 'resolved' | 'closed'
export type TicketPriority = 'low' | 'medium' | 'high' | 'critical'

export interface Ticket {
  id: number
  title: string
  description: string
  status: TicketStatus
  priority: TicketPriority
  category_id: number
  created_by: number
  assigned_to: number | null
  assigned_team_id: number | null
  created_at: string
  updated_at: string
  resolved_at: string | null
  closed_at: string | null
}

export type Role = 'admin' | 'agent' | 'customer'

export interface UserAdmin {
  id: number
  name: string
  email: string
  role: Role
  is_active: boolean
  created_at: string
}

export interface CreateUserInput {
  name: string
  email: string
  password: string
  role: Role
}

/** Visão mínima de um usuário atribuível a tickets (staff ativo). */
export interface Assignee {
  id: number
  name: string
  role: Role
}

export interface Category {
  id: number
  name: string
}

export interface Comment {
  id: number
  ticket_id: number
  author_id: number
  body: string
  is_internal: boolean
  created_at: string
}

export interface CreateTicketInput {
  title: string
  description: string
  category_id: number
  priority?: TicketPriority
}

/** Filtros opcionais da listagem de tickets; combináveis entre si. */
export interface TicketFilter {
  status?: TicketStatus
  priority?: TicketPriority
  assigned_to?: number
  q?: string
}

export interface UpdateTicketInput {
  title?: string
  description?: string
  status?: TicketStatus
  priority?: TicketPriority
  category_id?: number
  assigned_to?: number
}

/** Erro de API com o código estável retornado pelo backend (RNF13). */
export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
  token?: string
}

async function request<T>(path: string, { method = 'GET', body, token }: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`

  let res: Response
  try {
    res = await fetch(BASE + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'network_error', 'não foi possível conectar ao servidor')
  }

  if (res.status === 204) return undefined as T

  const data = await res.json().catch(() => null)
  if (!res.ok) {
    const code = data?.error?.code ?? 'unknown'
    const message = data?.error?.message ?? 'ocorreu um erro inesperado'
    throw new ApiError(res.status, code, message)
  }
  return data as T
}

export const api = {
  register: (input: RegisterInput) => request<AuthResult>('/auth/register', { method: 'POST', body: input }),
  login: (input: LoginInput) => request<AuthResult>('/auth/login', { method: 'POST', body: input }),
  refresh: (refreshToken: string) =>
    request<AuthResult>('/auth/refresh', { method: 'POST', body: { refresh_token: refreshToken } }),
  logout: (refreshToken: string) =>
    request<void>('/auth/logout', { method: 'POST', body: { refresh_token: refreshToken } }),
  me: (token: string) => request<{ user: User }>('/me', { token }),

  listUsers: (token: string) =>
    request<{ users: UserAdmin[] }>('/users', { token }).then((r) => r.users),
  createUser: (token: string, input: CreateUserInput) =>
    request<UserAdmin>('/users', { method: 'POST', body: input, token }),
  deactivateUser: (token: string, id: number) =>
    request<void>(`/users/${id}`, { method: 'DELETE', token }),
  listAssignees: (token: string) =>
    request<{ users: Assignee[] }>('/assignees', { token }).then((r) => r.users),

  listCategories: (token: string) =>
    request<{ categories: Category[] }>('/categories', { token }).then((r) => r.categories),
  createCategory: (token: string, name: string) =>
    request<Category>('/categories', { method: 'POST', body: { name }, token }),

  listTickets: (token: string, filter: TicketFilter = {}) => {
    const params = new URLSearchParams()
    for (const [key, value] of Object.entries(filter)) {
      if (value !== undefined && value !== '') params.set(key, String(value))
    }
    const qs = params.toString()
    return request<{ tickets: Ticket[] }>(`/tickets${qs ? `?${qs}` : ''}`, { token }).then((r) => r.tickets)
  },
  getTicket: (token: string, id: number) => request<Ticket>(`/tickets/${id}`, { token }),
  createTicket: (token: string, input: CreateTicketInput) =>
    request<Ticket>('/tickets', { method: 'POST', body: input, token }),
  updateTicket: (token: string, id: number, input: UpdateTicketInput) =>
    request<Ticket>(`/tickets/${id}`, { method: 'PATCH', body: input, token }),

  listComments: (token: string, ticketId: number) =>
    request<{ comments: Comment[] }>(`/tickets/${ticketId}/comments`, { token }).then((r) => r.comments),
  addComment: (token: string, ticketId: number, body: string, isInternal = false) =>
    request<Comment>(`/tickets/${ticketId}/comments`, {
      method: 'POST',
      body: { body, is_internal: isInternal },
      token,
    }),
}
