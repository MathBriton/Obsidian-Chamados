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
  /** Executa uma chamada autenticada injetando o access token. Em caso de 401,
   * rotaciona via refresh token e tenta uma vez mais; se falhar, encerra a sessão. */
  authCall: <T>(fn: (token: string) => Promise<T>) => Promise<T>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

/** Acessa o contexto de autenticação. Lança se usado fora do AuthProvider. */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth deve ser usado dentro de <AuthProvider>')
  return ctx
}
