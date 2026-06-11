import { useCallback, useState, type FormEvent } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { ApiError, api } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { TextField } from '../components/TextField'

export function CategoriesPage() {
  const { authCall, user } = useAuth()
  const loader = useCallback(() => authCall((t) => api.listCategories(t)), [authCall])
  const { data: categories, loading, error, reload } = useAsync(loader)

  // Defesa em profundidade: a rota é exibida só para admin, mas reforçamos aqui.
  if (user && user.role !== 'admin') return <Navigate to="/" replace />

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold text-slate-900">Categorias</h1>

      <CreateCategoryForm onCreated={reload} />

      {loading && <p className="text-slate-500">Carregando categorias…</p>}
      {error && <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}

      {categories && categories.length === 0 && (
        <div className="mt-6 rounded-2xl border border-dashed border-slate-300 bg-white p-10 text-center text-slate-500">
          Nenhuma categoria ainda. Crie a primeira para permitir a abertura de chamados.
        </div>
      )}

      {categories && categories.length > 0 && (
        <ul className="mt-6 space-y-2">
          {categories.map((c) => (
            <li
              key={c.id}
              className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3"
            >
              <p className="truncate font-medium text-slate-900">{c.name}</p>
              <span className="shrink-0 text-xs text-slate-400">#{c.id}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function CreateCategoryForm({ onCreated }: { onCreated: () => void }) {
  const { authCall } = useAuth()
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await authCall((t) => api.createCategory(t, name.trim()))
      setName('')
      onCreated()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao criar categoria')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <h2 className="mb-4 font-medium text-slate-800">Nova categoria</h2>
      {error && (
        <div role="alert" className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      )}
      <div className="flex flex-wrap items-end gap-3">
        <div className="min-w-64 flex-1">
          <TextField label="Nome" value={name} onChange={setName} required placeholder="ex.: Infraestrutura" />
        </div>
        <button
          type="submit"
          disabled={submitting}
          className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-violet-700 disabled:opacity-60"
        >
          {submitting ? 'Criando…' : 'Criar categoria'}
        </button>
      </div>
    </form>
  )
}
