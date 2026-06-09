import { useCallback, useState, type FormEvent } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { ApiError, api, type Role } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { roleLabels } from '../lib/labels'
import { TextField } from '../components/TextField'

const creatableRoles: Role[] = ['customer', 'agent', 'admin']

export function UsersPage() {
  const { authCall, user } = useAuth()
  const loader = useCallback(() => authCall((t) => api.listUsers(t)), [authCall])
  const { data: users, loading, error, reload } = useAsync(loader)

  // Defesa em profundidade: a rota é exibida só para admin, mas reforçamos aqui.
  if (user && user.role !== 'admin') return <Navigate to="/" replace />

  async function handleDeactivate(id: number) {
    await authCall((t) => api.deactivateUser(t, id))
    reload()
  }

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold text-slate-900">Usuários</h1>

      <CreateUserForm onCreated={reload} />

      {loading && <p className="text-slate-500">Carregando usuários…</p>}
      {error && <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}

      {users && (
        <ul className="mt-6 space-y-2">
          {users.map((u) => (
            <li
              key={u.id}
              className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3"
            >
              <div className="min-w-0">
                <p className="truncate font-medium text-slate-900">
                  {u.name}
                  {!u.is_active && <span className="ml-2 text-xs font-normal text-slate-400">(inativo)</span>}
                </p>
                <p className="truncate text-sm text-slate-500">{u.email}</p>
              </div>
              <div className="flex shrink-0 items-center gap-3">
                <span className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600">
                  {roleLabels[u.role]}
                </span>
                {u.is_active && u.id !== user?.id && (
                  <button
                    onClick={() => void handleDeactivate(u.id)}
                    className="rounded-lg border border-red-200 px-2.5 py-1 text-xs font-medium text-red-600 transition hover:bg-red-50"
                  >
                    Desativar
                  </button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function CreateUserForm({ onCreated }: { onCreated: () => void }) {
  const { authCall } = useAuth()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('customer')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await authCall((t) => api.createUser(t, { name, email, password, role }))
      setName('')
      setEmail('')
      setPassword('')
      setRole('customer')
      onCreated()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao criar usuário')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <h2 className="mb-4 font-medium text-slate-800">Novo usuário</h2>
      {error && (
        <div role="alert" className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      )}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <TextField label="Nome" value={name} onChange={setName} required />
        <TextField label="E-mail" type="email" value={email} onChange={setEmail} required autoComplete="off" />
        <TextField
          label="Senha"
          type="password"
          value={password}
          onChange={setPassword}
          required
          autoComplete="new-password"
          placeholder="mínimo 8 caracteres"
        />
        <div className="flex flex-col gap-1">
          <label htmlFor="role" className="text-sm font-medium text-slate-700">
            Papel
          </label>
          <select
            id="role"
            value={role}
            onChange={(e) => setRole(e.target.value as Role)}
            className="rounded-lg border border-slate-300 px-3 py-2 text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
          >
            {creatableRoles.map((r) => (
              <option key={r} value={r}>
                {roleLabels[r]}
              </option>
            ))}
          </select>
        </div>
      </div>
      <button
        type="submit"
        disabled={submitting}
        className="mt-4 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-violet-700 disabled:opacity-60"
      >
        {submitting ? 'Criando…' : 'Criar usuário'}
      </button>
    </form>
  )
}
