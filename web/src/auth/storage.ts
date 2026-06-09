// Persistência da sessão no localStorage. Guarda tokens + identidade para
// sobreviver a reloads. (Em produção real, refresh tokens em cookie HttpOnly
// seriam mais seguros; aqui mantemos simples para o portfólio.)
import type { AuthResult, Tenant, User } from '../lib/api'

const KEY = 'obsidian.auth'

export interface Session {
  accessToken: string
  refreshToken: string
  user: User
  tenant: Tenant
}

export function loadSession(): Session | null {
  const raw = localStorage.getItem(KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as Session
  } catch {
    localStorage.removeItem(KEY)
    return null
  }
}

export function saveSession(r: AuthResult): Session {
  const session: Session = {
    accessToken: r.access_token,
    refreshToken: r.refresh_token,
    user: r.user,
    tenant: r.tenant,
  }
  localStorage.setItem(KEY, JSON.stringify(session))
  return session
}

export function clearSession(): void {
  localStorage.removeItem(KEY)
}
