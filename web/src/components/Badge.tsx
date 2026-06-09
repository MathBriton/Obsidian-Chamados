import type { TicketPriority, TicketStatus } from '../lib/api'
import { priorityLabels, statusLabels } from '../lib/labels'

function Badge({ label, cls }: { label: string; cls: string }) {
  return <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${cls}`}>{label}</span>
}

export function StatusBadge({ status }: { status: TicketStatus }) {
  const s = statusLabels[status]
  return <Badge label={s.label} cls={s.cls} />
}

export function PriorityBadge({ priority }: { priority: TicketPriority }) {
  const p = priorityLabels[priority]
  return <Badge label={p.label} cls={p.cls} />
}
