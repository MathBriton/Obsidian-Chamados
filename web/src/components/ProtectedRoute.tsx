import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/context'

/** Protege rotas que exigem sessão. Enquanto restaura a sessão, mostra um
 * placeholder; sem usuário, redireciona para /login. */
export function ProtectedRoute() {
  const { user, loading } = useAuth()

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center text-slate-500">
        Carregando…
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
