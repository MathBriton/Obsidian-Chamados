import { useCallback, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { ApiError, api, type TicketPriority, type TicketStatus } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { priorityLabels, priorityOrder, statusLabels, statusOrder } from '../lib/labels'
import { PriorityBadge, StatusBadge } from '../components/Badge'

export function TicketDetailPage() {
  const { id } = useParams()
  const ticketId = Number(id)
  const { authCall, user } = useAuth()
  const isStaff = user?.role === 'admin' || user?.role === 'agent'

  const loadTicket = useCallback(() => authCall((t) => api.getTicket(t, ticketId)), [authCall, ticketId])
  const loadComments = useCallback(() => authCall((t) => api.listComments(t, ticketId)), [authCall, ticketId])
  const { data: ticket, loading, error, setData: setTicket } = useAsync(loadTicket)
  const { data: comments, reload: reloadComments } = useAsync(loadComments)

  async function patch(input: { status?: TicketStatus; priority?: TicketPriority; assigned_to?: number }) {
    const updated = await authCall((t) => api.updateTicket(t, ticketId, input))
    setTicket(updated)
  }

  if (loading) return <p className="text-slate-500">Carregando…</p>
  if (error)
    return (
      <div>
        <BackLink />
        <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
      </div>
    )
  if (!ticket) return null

  return (
    <div className="mx-auto max-w-3xl">
      <BackLink />

      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold text-slate-900">{ticket.title}</h1>
            <p className="mt-0.5 text-xs text-slate-400">
              #{ticket.id} · aberto em {new Date(ticket.created_at).toLocaleString('pt-BR')}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <PriorityBadge priority={ticket.priority} />
            <StatusBadge status={ticket.status} />
          </div>
        </div>

        <p className="mt-4 whitespace-pre-wrap text-slate-700">{ticket.description}</p>

        {isStaff && (
          <div className="mt-6 grid grid-cols-1 gap-4 border-t border-slate-100 pt-4 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-sm font-medium text-slate-700">
              Status
              <select
                value={ticket.status}
                onChange={(e) => void patch({ status: e.target.value as TicketStatus })}
                className="rounded-lg border border-slate-300 px-3 py-2 font-normal text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
              >
                {statusOrder.map((s) => (
                  <option key={s} value={s}>
                    {statusLabels[s].label}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-sm font-medium text-slate-700">
              Prioridade
              <select
                value={ticket.priority}
                onChange={(e) => void patch({ priority: e.target.value as TicketPriority })}
                className="rounded-lg border border-slate-300 px-3 py-2 font-normal text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
              >
                {priorityOrder.map((p) => (
                  <option key={p} value={p}>
                    {priorityLabels[p].label}
                  </option>
                ))}
              </select>
            </label>
            <AssigneeSelect value={ticket.assigned_to} onChange={(assignedTo) => void patch({ assigned_to: assignedTo })} />
          </div>
        )}
      </div>

      <section className="mt-8">
        <h2 className="mb-3 text-lg font-semibold text-slate-900">Comentários</h2>

        <ul className="space-y-3">
          {comments?.map((c) => (
            <li
              key={c.id}
              className={`rounded-xl border px-4 py-3 ${
                c.is_internal ? 'border-amber-200 bg-amber-50' : 'border-slate-200 bg-white'
              }`}
            >
              <div className="mb-1 flex items-center gap-2 text-xs text-slate-400">
                <span>Usuário #{c.author_id}</span>
                <span>·</span>
                <span>{new Date(c.created_at).toLocaleString('pt-BR')}</span>
                {c.is_internal && (
                  <span className="rounded bg-amber-200 px-1.5 py-0.5 font-medium text-amber-800">nota interna</span>
                )}
              </div>
              <p className="whitespace-pre-wrap text-sm text-slate-700">{c.body}</p>
            </li>
          ))}
          {comments && comments.length === 0 && <p className="text-sm text-slate-400">Nenhum comentário ainda.</p>}
        </ul>

        <CommentForm ticketId={ticketId} isStaff={isStaff} onAdded={reloadComments} />
      </section>
    </div>
  )
}

/** Select de responsável (staff). Popula com os usuários atribuíveis do tenant.
 * Não permite "desatribuir" (o backend não suporta limpar assigned_to). */
function AssigneeSelect({ value, onChange }: { value: number | null; onChange: (id: number) => void }) {
  const { authCall } = useAuth()
  const loader = useCallback(() => authCall((t) => api.listAssignees(t)), [authCall])
  const { data: assignees } = useAsync(loader)

  return (
    <label className="flex flex-col gap-1 text-sm font-medium text-slate-700">
      Responsável
      <select
        value={value ?? ''}
        onChange={(e) => {
          const id = Number(e.target.value)
          if (id > 0) onChange(id)
        }}
        className="rounded-lg border border-slate-300 px-3 py-2 font-normal text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
      >
        <option value="">— Sem responsável —</option>
        {assignees?.map((a) => (
          <option key={a.id} value={a.id}>
            {a.name}
          </option>
        ))}
      </select>
    </label>
  )
}

function BackLink() {
  return (
    <Link to="/" className="mb-4 inline-block text-sm text-slate-500 hover:underline">
      ← Voltar aos chamados
    </Link>
  )
}

function CommentForm({ ticketId, isStaff, onAdded }: { ticketId: number; isStaff: boolean; onAdded: () => void }) {
  const { authCall } = useAuth()
  const [body, setBody] = useState('')
  const [internal, setInternal] = useState(false)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!body.trim()) return
    setSubmitting(true)
    setError('')
    try {
      await authCall((t) => api.addComment(t, ticketId, body.trim(), internal))
      setBody('')
      setInternal(false)
      onAdded()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao comentar')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="mt-4 rounded-xl border border-slate-200 bg-white p-4">
      {error && <p className="mb-2 text-sm text-red-600">{error}</p>}
      <textarea
        aria-label="Novo comentário"
        value={body}
        onChange={(e) => setBody(e.target.value)}
        rows={3}
        placeholder="Escreva um comentário…"
        className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
      />
      <div className="mt-2 flex items-center justify-between">
        {isStaff ? (
          <label className="flex items-center gap-2 text-sm text-slate-600">
            <input type="checkbox" checked={internal} onChange={(e) => setInternal(e.target.checked)} />
            Nota interna (só a equipe vê)
          </label>
        ) : (
          <span />
        )}
        <button
          type="submit"
          disabled={submitting}
          className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-violet-700 disabled:opacity-60"
        >
          {submitting ? 'Enviando…' : 'Comentar'}
        </button>
      </div>
    </form>
  )
}
