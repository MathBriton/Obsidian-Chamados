import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { ApiError, api, type TicketPriority, type TicketStatus } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { priorityLabels, priorityOrder, statusLabels, statusOrder } from '../lib/labels'
import { PriorityBadge, StatusBadge } from '../components/Badge'

/** Atraso do debounce da busca por texto, para não consultar a cada tecla. */
const SEARCH_DEBOUNCE_MS = 300

/** Tamanho de página da listagem (o backend limita em 50). */
const PAGE_SIZE = 20

export function TicketsPage() {
  const { authCall } = useAuth()
  const [status, setStatus] = useState<TicketStatus | ''>('')
  const [priority, setPriority] = useState<TicketPriority | ''>('')
  const [search, setSearch] = useState('')
  const [q, setQ] = useState('')

  // exhausted marca que o servidor devolveu uma página incompleta (fim da
  // lista); loadingMore/moreError são o estado do "Carregar mais".
  const [exhausted, setExhausted] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [moreError, setMoreError] = useState('')

  useEffect(() => {
    const id = setTimeout(() => setQ(search.trim()), SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(id)
  }, [search])

  const loader = useCallback(() => {
    setExhausted(false)
    setMoreError('')
    return authCall((t) =>
      api.listTickets(t, {
        status: status || undefined,
        priority: priority || undefined,
        q: q || undefined,
        limit: PAGE_SIZE,
        offset: 0,
      }),
    )
  }, [authCall, status, priority, q])
  const { data: tickets, loading, error, setData } = useAsync(loader)

  // Página cheia sugere que há mais; uma incompleta encerra a paginação. No
  // caso raro de o total ser múltiplo exato, a próxima página vem vazia e
  // marca exhausted.
  const hasMore = !exhausted && !!tickets && tickets.length > 0 && tickets.length % PAGE_SIZE === 0

  async function loadMore() {
    if (!tickets) return
    setLoadingMore(true)
    setMoreError('')
    try {
      const page = await authCall((t) =>
        api.listTickets(t, {
          status: status || undefined,
          priority: priority || undefined,
          q: q || undefined,
          limit: PAGE_SIZE,
          offset: tickets.length,
        }),
      )
      setData([...tickets, ...page])
      if (page.length < PAGE_SIZE) setExhausted(true)
    } catch (err) {
      setMoreError(err instanceof ApiError ? err.message : 'falha ao carregar mais chamados')
    } finally {
      setLoadingMore(false)
    }
  }

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
        <>
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

          {moreError && (
            <p className="mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{moreError}</p>
          )}
          {hasMore && (
            <div className="mt-4 text-center">
              <button
                onClick={() => void loadMore()}
                disabled={loadingMore}
                className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-100 disabled:opacity-60"
              >
                {loadingMore ? 'Carregando…' : 'Carregar mais'}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
