import { useCallback } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { api } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { PriorityBadge, StatusBadge } from '../components/Badge'

export function TicketsPage() {
  const { authCall } = useAuth()
  const loader = useCallback(() => authCall((t) => api.listTickets(t)), [authCall])
  const { data: tickets, loading, error } = useAsync(loader)

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-slate-900">Chamados</h1>
        <Link
          to="/tickets/new"
          className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-violet-700"
        >
          + Novo chamado
        </Link>
      </div>

      {loading && <p className="text-slate-500">Carregando chamados…</p>}
      {error && <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}

      {tickets && tickets.length === 0 && (
        <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-10 text-center text-slate-500">
          Nenhum chamado ainda. Crie o primeiro!
        </div>
      )}

      {tickets && tickets.length > 0 && (
        <ul className="space-y-2">
          {tickets.map((ticket) => (
            <li key={ticket.id}>
              <Link
                to={`/tickets/${ticket.id}`}
                className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3 transition hover:border-violet-300 hover:shadow-sm"
              >
                <div className="min-w-0">
                  <p className="truncate font-medium text-slate-900">{ticket.title}</p>
                  <p className="text-xs text-slate-400">#{ticket.id}</p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <PriorityBadge priority={ticket.priority} />
                  <StatusBadge status={ticket.status} />
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
