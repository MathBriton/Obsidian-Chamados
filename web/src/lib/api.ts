// Cliente HTTP da API Obsidian Chamados. Em dev, a base é vazia e o proxy do
// Vite encaminha para o backend Go; em produção, defina VITE_API_URL.

const BASE = import.meta.env.VITE_API_URL ?? ''

export interface User {
  id: number
  name: string
  email: string
  role: string
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
}
