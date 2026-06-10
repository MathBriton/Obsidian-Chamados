import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { api, type TicketPriority, type TicketStatus } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { priorityLabels, priorityOrder, statusLabels, statusOrder } from '../lib/labels'
import { PriorityBadge, StatusBadge } from '../components/Badge'

/** Atraso do debounce da busca por texto, para não consultar a cada tecla. */
const SEARCH_DEBOUNCE_MS = 300

export function TicketsPage() {
  const { authCall } = useAuth()
  const [status, setStatus] = useState<TicketStatus | ''>('')
  const [priority, setPriority] = useState<TicketPriority | ''>('')
  const [search, setSearch] = useState('')
  const [q, setQ] = useState('')

  useEffect(() => {
    const id = setTimeout(() => setQ(search.trim()), SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(id)
  }, [search])

  const loader = useCallback(
    () =>
      authCall((t) =>
        api.listTickets(t, {
          status: status || undefined,
          priority: priority || undefined,
          q: q || undefined,
        }),
      ),
    [authCall, status, priority, q],
  )
  const { data: tickets, loading, error } = useAsync(loader)

  const hasFilter = status !== '' || priority !== '' || q !== ''

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

      <div className="mb-4 flex flex-wrap gap-2">
        <input
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Buscar por título ou descrição…"
          aria-label="Buscar chamados"
          className="w-full max-w-xs rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200 sm:w-auto sm:flex-1"
        />
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value as TicketStatus | '')}
          aria-label="Filtrar por status"
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
        >
          <option value="">Todos os status</option>
          {statusOrder.map((s) => (
            <option key={s} value={s}>
              {statusLabels[s].label}
            </option>
          ))}
        </select>
        <select
          value={priority}
          onChange={(e) => setPriority(e.target.value as TicketPriority | '')}
          aria-label="Filtrar por prioridade"
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
        >
          <option value="">Todas as prioridades</option>
          {priorityOrder.map((p) => (
            <option key={p} value={p}>
              {priorityLabels[p].label}
            </option>
          ))}
        </select>
      </div>

      {loading && <p className="text-slate-500">Carregando chamados…</p>}
      {error && <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}

      {tickets && tickets.length === 0 && (
        <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-10 text-center text-slate-500">
          {hasFilter ? 'Nenhum chamado corresponde aos filtros.' : 'Nenhum chamado ainda. Crie o primeiro!'}
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
