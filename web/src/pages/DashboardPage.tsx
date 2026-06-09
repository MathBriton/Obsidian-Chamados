import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/context'

const roleLabels: Record<string, string> = {
  admin: 'Administrador',
  agent: 'Atendente',
  customer: 'Cliente',
}

export function DashboardPage() {
  const { user, tenant, logout } = useAuth()
  const navigate = useNavigate()

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-4">
          <div>
            <h1 className="text-lg font-semibold text-slate-900">Obsidian Chamados</h1>
            <p className="text-sm text-slate-500">{tenant?.name}</p>
          </div>
          <button
            onClick={handleLogout}
            className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 transition hover:bg-slate-100"
          >
            Sair
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 py-10">
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-xl font-semibold text-slate-900">Olá, {user?.name} 👋</h2>
          <p className="mt-1 text-sm text-slate-500">Sua sessão está ativa.</p>

          <dl className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Info label="E-mail" value={user?.email ?? '—'} />
            <Info label="Papel" value={user ? (roleLabels[user.role] ?? user.role) : '—'} />
            <Info label="Empresa" value={tenant?.name ?? '—'} />
            <Info label="Slug" value={tenant?.slug ?? '—'} />
          </dl>

          <p className="mt-8 text-sm text-slate-400">
            Em breve: gestão de tickets, categorias e comentários.
          </p>
        </div>
      </main>
    </div>
  )
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg bg-slate-50 px-4 py-3">
      <dt className="text-xs uppercase tracking-wide text-slate-400">{label}</dt>
      <dd className="mt-0.5 font-medium text-slate-800">{value}</dd>
    </div>
  )
}
