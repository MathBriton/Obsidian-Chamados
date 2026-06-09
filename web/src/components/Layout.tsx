import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/context'

const roleLabels: Record<string, string> = {
  admin: 'Administrador',
  agent: 'Atendente',
  customer: 'Cliente',
}

/** Casca das páginas autenticadas: cabeçalho com empresa, usuário e logout. */
export function Layout() {
  const { user, tenant, logout } = useAuth()
  const navigate = useNavigate()

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-3">
          <Link to="/" className="flex flex-col">
            <span className="text-lg font-semibold text-slate-900">Obsidian Chamados</span>
            <span className="text-xs text-slate-500">{tenant?.name}</span>
          </Link>
          <div className="flex items-center gap-4">
            <div className="text-right">
              <p className="text-sm font-medium text-slate-800">{user?.name}</p>
              <p className="text-xs text-slate-500">{user ? (roleLabels[user.role] ?? user.role) : ''}</p>
            </div>
            <button
              onClick={handleLogout}
              className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 transition hover:bg-slate-100"
            >
              Sair
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 py-8">
        <Outlet />
      </main>
    </div>
  )
}
