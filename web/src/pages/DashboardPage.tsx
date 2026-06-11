import { useCallback } from 'react'
import { useAuth } from '../auth/context'
import { api, type TicketStats } from '../lib/api'
import { useAsync } from '../lib/useAsync'
import { priorityLabels, priorityOrder, statusLabels, statusOrder } from '../lib/labels'

/** Status considerados "ativos" (demandam ação de alguém). */
const activeStatuses = ['open', 'in_progress', 'waiting_customer'] as const

export function DashboardPage() {
  const { authCall, user } = useAuth()
  const loader = useCallback(() => authCall((t) => api.getStats(t)), [authCall])
  const { data: stats, loading, error } = useAsync(loader)

  const isCustomer = user?.role === 'customer'

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-slate-900">Dashboard</h1>
        <p className="text-sm text-slate-500">
          {isCustomer ? 'Resumo dos seus chamados.' : 'Resumo de todos os chamados da empresa.'}
        </p>
      </div>

      {loading && <p className="text-slate-500">Carregando métricas…</p>}
      {error && <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}

      {stats && (
        <>
          <SummaryCards stats={stats} />
          <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
            <BreakdownCard
              title="Por status"
              rows={statusOrder.map((s) => ({ key: s, label: statusLabels[s].label, cls: statusLabels[s].cls, count: stats.by_status[s] }))}
              total={stats.total}
            />
            <BreakdownCard
              title="Por prioridade"
              rows={priorityOrder.map((p) => ({ key: p, label: priorityLabels[p].label, cls: priorityLabels[p].cls, count: stats.by_priority[p] }))}
              total={stats.total}
            />
          </div>
        </>
      )}
    </div>
  )
}

function SummaryCards({ stats }: { stats: TicketStats }) {
  const active = activeStatuses.reduce((sum, s) => sum + stats.by_status[s], 0)
  const closed = stats.by_status.resolved + stats.by_status.closed

  const cards = [
    { label: 'Total de chamados', value: stats.total },
    { label: 'Chamados ativos', value: active },
    { label: 'Ativos sem responsável', value: stats.unassigned_active },
    { label: 'Encerrados', value: closed },
  ]

  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      {cards.map((c) => (
        <div key={c.label} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <p className="text-sm text-slate-500">{c.label}</p>
          <p aria-label={c.label} className="mt-1 text-3xl font-semibold text-slate-900">
            {c.value}
          </p>
        </div>
      ))}
    </div>
  )
}

interface BreakdownRow {
  key: string
  label: string
  cls: string
  count: number
}

function BreakdownCard({ title, rows, total }: { title: string; rows: BreakdownRow[]; total: number }) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <h2 className="mb-4 font-medium text-slate-800">{title}</h2>
      <ul className="space-y-3">
        {rows.map((row) => (
          <li key={row.key} className="flex items-center gap-3">
            <span className={`w-40 shrink-0 rounded-full px-2.5 py-0.5 text-center text-xs font-medium ${row.cls}`}>
              {row.label}
            </span>
            <div className="h-2 flex-1 overflow-hidden rounded-full bg-slate-100">
              <div
                className="h-full rounded-full bg-violet-500"
                style={{ width: total > 0 ? `${(row.count / total) * 100}%` : '0%' }}
              />
            </div>
            <span className="w-8 shrink-0 text-right text-sm font-medium text-slate-700">{row.count}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
