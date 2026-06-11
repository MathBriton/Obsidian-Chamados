import { useCallback, useState, type FormEvent } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { ApiError, api, type Assignee, type Team } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { roleLabels } from '../lib/labels'
import { TextField } from '../components/TextField'

export function TeamsPage() {
  const { authCall, user } = useAuth()
  const loadTeams = useCallback(() => authCall((t) => api.listTeams(t)), [authCall])
  const loadAssignees = useCallback(() => authCall((t) => api.listAssignees(t)), [authCall])
  const { data: teams, loading, error, reload } = useAsync(loadTeams)
  const { data: assignees } = useAsync(loadAssignees)

  // Defesa em profundidade: a rota é exibida só para admin, mas reforçamos aqui.
  if (user && user.role !== 'admin') return <Navigate to="/" replace />

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold text-slate-900">Equipes</h1>

      <CreateTeamForm onCreated={reload} />

      {loading && <p className="text-slate-500">Carregando equipes…</p>}
      {error && <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}

      {teams && teams.length === 0 && (
        <div className="mt-6 rounded-2xl border border-dashed border-slate-300 bg-white p-10 text-center text-slate-500">
          Nenhuma equipe ainda. Crie a primeira para organizar o atendimento.
        </div>
      )}

      {teams && teams.length > 0 && (
        <ul className="mt-6 space-y-4">
          {teams.map((team) => (
            <TeamCard key={team.id} team={team} assignees={assignees ?? []} onChanged={reload} />
          ))}
        </ul>
      )}
    </div>
  )
}

function TeamCard({ team, assignees, onChanged }: { team: Team; assignees: Assignee[]; onChanged: () => void }) {
  const { authCall } = useAuth()
  const [error, setError] = useState('')

  const memberIds = new Set(team.members.map((m) => m.id))
  const addable = assignees.filter((a) => !memberIds.has(a.id))

  async function addMember(userId: number) {
    setError('')
    try {
      await authCall((t) => api.addTeamMember(t, team.id, userId))
      onChanged()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao adicionar membro')
    }
  }

  async function removeMember(userId: number) {
    setError('')
    try {
      await authCall((t) => api.removeTeamMember(t, team.id, userId))
      onChanged()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao remover membro')
    }
  }

  return (
    <li className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex items-center justify-between gap-4">
        <p className="font-medium text-slate-900">{team.name}</p>
        <span className="shrink-0 text-xs text-slate-400">#{team.id}</span>
      </div>

      {error && (
        <p role="alert" className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </p>
      )}

      {team.members.length === 0 ? (
        <p className="mt-3 text-sm text-slate-400">Nenhum membro ainda.</p>
      ) : (
        <ul className="mt-3 flex flex-wrap gap-2">
          {team.members.map((m) => (
            <li
              key={m.id}
              className="flex items-center gap-2 rounded-full bg-slate-100 py-1 pl-3 pr-1.5 text-sm text-slate-700"
            >
              <span>{m.name}</span>
              <span className="text-xs text-slate-400">{roleLabels[m.role]}</span>
              <button
                type="button"
                aria-label={`Remover ${m.name}`}
                onClick={() => void removeMember(m.id)}
                className="rounded-full px-1.5 text-slate-400 transition hover:bg-slate-200 hover:text-slate-700"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}

      {addable.length > 0 && (
        <div className="mt-3">
          <label className="flex max-w-xs flex-col gap-1 text-sm font-medium text-slate-700">
            Adicionar membro
            <select
              value=""
              onChange={(e) => {
                const id = Number(e.target.value)
                if (id > 0) void addMember(id)
              }}
              className="rounded-lg border border-slate-300 px-3 py-2 font-normal text-slate-900 outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-200"
            >
              <option value="">Selecione…</option>
              {addable.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </label>
        </div>
      )}
    </li>
  )
}

function CreateTeamForm({ onCreated }: { onCreated: () => void }) {
  const { authCall } = useAuth()
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await authCall((t) => api.createTeam(t, name.trim()))
      setName('')
      onCreated()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao criar equipe')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <h2 className="mb-4 font-medium text-slate-800">Nova equipe</h2>
      {error && (
        <div role="alert" className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      )}
      <div className="flex flex-wrap items-end gap-3">
        <div className="min-w-64 flex-1">
          <TextField label="Nome" value={name} onChange={setName} required placeholder="ex.: Suporte N1" />
        </div>
        <button
          type="submit"
          disabled={submitting}
          className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-violet-700 disabled:opacity-60"
        >
          {submitting ? 'Criando…' : 'Criar equipe'}
        </button>
      </div>
    </form>
  )
}
