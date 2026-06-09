import { useCallback, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { ApiError, api, type TicketPriority } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { priorityLabels, priorityOrder } from '../lib/labels'
import { TextField } from '../components/TextField'

export function NewTicketPage() {
  const { authCall, user } = useAuth()
  const navigate = useNavigate()

  const loadCategories = useCallback(() => authCall((t) => api.listCategories(t)), [authCall])
  const { data: categories, loading: loadingCategories, reload } = useAsync(loadCategories)

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<TicketPriority>('medium')
  const [categoryId, setCategoryId] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const isAdmin = user?.role === 'admin'

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (!categoryId) {
      setError('selecione uma categoria')
      return
    }
    setSubmitting(true)
    try {
      const ticket = await authCall((t) =>
        api.createTicket(t, { title, description, category_id: Number(categoryId), priority }),
      )
      navigate(`/tickets/${ticket.id}`, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao criar chamado')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-6 flex items-center gap-2 text-sm text-slate-500">
        <Link to="/" className="hover:underline">
          Chamados
        </Link>
        <span>/</span>
        <span className="text-slate-700">Novo</span>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h1 className="mb-6 text-xl font-semibold text-slate-900">Abrir chamado</h1>

        {error && (
          <div role="alert" className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {error}
          </div>
        )}

        {!loadingCategories && categories && categories.length === 0 && (
          <CategoryEmptyState isAdmin={isAdmin} onCreated={reload} />
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <TextField label="Título" value={title} onChange={setTitle} required />

          <div className="flex flex-col gap-1">
            <label htmlFor="description" className="text-sm font-medium text-slate-700">
              Descrição
            </label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              required
              rows={4}
              className="rounded-lg border border-slate-300 px-3 py-2 text-slate-900 outline-none transition focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1">
              <label htmlFor="category" className="text-sm font-medium text-slate-700">
                Categoria
              </label>
              <select
                id="category"
                value={categoryId}
                onChange={(e) => setCategoryId(e.target.value)}
                className="rounded-lg border border-slate-300 px-3 py-2 text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
              >
                <option value="">Selecione…</option>
                {categories?.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex flex-col gap-1">
              <label htmlFor="priority" className="text-sm font-medium text-slate-700">
                Prioridade
              </label>
              <select
                id="priority"
                value={priority}
                onChange={(e) => setPriority(e.target.value as TicketPriority)}
                className="rounded-lg border border-slate-300 px-3 py-2 text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
              >
                {priorityOrder.map((p) => (
                  <option key={p} value={p}>
                    {priorityLabels[p].label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="mt-2 flex gap-3">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-violet-600 px-4 py-2 font-medium text-white transition hover:bg-violet-700 disabled:opacity-60"
            >
              {submitting ? 'Criando…' : 'Criar chamado'}
            </button>
            <Link to="/" className="rounded-lg border border-slate-300 px-4 py-2 font-medium text-slate-700 hover:bg-slate-100">
              Cancelar
            </Link>
          </div>
        </form>
      </div>
    </div>
  )
}

/** Aviso quando não há categorias. Admin pode criar uma ali mesmo. */
function CategoryEmptyState({ isAdmin, onCreated }: { isAdmin: boolean; onCreated: () => void }) {
  const { authCall } = useAuth()
  const [name, setName] = useState('')
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')

  if (!isAdmin) {
    return (
      <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800">
        Ainda não há categorias. Peça a um administrador para cadastrá-las.
      </div>
    )
  }

  async function handleCreate() {
    if (!name.trim()) return
    setCreating(true)
    setError('')
    try {
      await authCall((t) => api.createCategory(t, name.trim()))
      setName('')
      onCreated()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao criar categoria')
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="mb-4 rounded-lg border border-slate-200 bg-slate-50 p-3">
      <p className="mb-2 text-sm text-slate-600">Nenhuma categoria ainda. Crie a primeira:</p>
      {error && <p className="mb-2 text-sm text-red-600">{error}</p>}
      <div className="flex gap-2">
        <input
          aria-label="Nome da categoria"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="ex.: Infraestrutura"
          className="flex-1 rounded-lg border border-slate-300 px-3 py-1.5 text-sm outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
        />
        <button
          type="button"
          onClick={handleCreate}
          disabled={creating}
          className="rounded-lg bg-slate-800 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-slate-900 disabled:opacity-60"
        >
          Criar
        </button>
      </div>
    </div>
  )
}
