import { createContext, useContext } from 'react'
import type { RegisterInput, Tenant, User } from '../lib/api'

export interface AuthContextValue {
  user: User | null
  tenant: Tenant | null
  accessToken: string | null
  /** true enquanto a sessão persistida está sendo restaurada/validada. */
  loading: boolean
  login: (slug: string, email: string, password: string) => Promise<void>
  register: (input: RegisterInput) => Promise<void>
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

/** Acessa o contexto de autenticação. Lança se usado fora do AuthProvider. */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth deve ser usado dentro de <AuthProvider>')
  return ctx
}
