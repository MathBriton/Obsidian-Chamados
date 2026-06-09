import { useEffect, useState, type ReactNode } from 'react'
import { ApiError, api, type RegisterInput } from '../lib/api'
import { AuthContext, type AuthContextValue } from './context'
import { clearSession, loadSession, saveSession, type Session } from './storage'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(() => loadSession())
  const [loading, setLoading] = useState(true)

  // Ao montar, restaura a sessão persistida validando o access token em /me.
  // Se expirou (401), tenta rotacionar via refresh; se falhar, encerra a sessão.
  useEffect(() => {
    let active = true
    async function restore() {
      const stored = loadSession()
      if (!stored) {
        if (active) setLoading(false)
        return
      }
      try {
        await api.me(stored.accessToken)
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          try {
            const refreshed = saveSession(await api.refresh(stored.refreshToken))
            if (active) setSession(refreshed)
          } catch {
            clearSession()
            if (active) setSession(null)
          }
        }
        // Erros de rede mantêm a sessão otimisticamente.
      } finally {
        if (active) setLoading(false)
      }
    }
    void restore()
    return () => {
      active = false
    }
  }, [])

  const login = async (slug: string, email: string, password: string) => {
    setSession(saveSession(await api.login({ slug, email, password })))
  }

  const register = async (input: RegisterInput) => {
    setSession(saveSession(await api.register(input)))
  }

  const logout = async () => {
    const current = session
    setSession(null)
    clearSession()
    if (current) {
      try {
        await api.logout(current.refreshToken)
      } catch {
        // logout é best-effort; o token expira no servidor de qualquer forma.
      }
    }
  }

  const value: AuthContextValue = {
    user: session?.user ?? null,
    tenant: session?.tenant ?? null,
    accessToken: session?.accessToken ?? null,
    loading,
    login,
    register,
    logout,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
