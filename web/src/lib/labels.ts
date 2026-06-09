import type { Role, TicketPriority, TicketStatus } from './api'

interface LabelStyle {
  label: string
  /** classes Tailwind do badge (texto + fundo) */
  cls: string
}

export const statusLabels: Record<TicketStatus, LabelStyle> = {
  open: { label: 'Aberto', cls: 'bg-blue-100 text-blue-700' },
  in_progress: { label: 'Em andamento', cls: 'bg-amber-100 text-amber-700' },
  waiting_customer: { label: 'Aguardando cliente', cls: 'bg-purple-100 text-purple-700' },
  resolved: { label: 'Resolvido', cls: 'bg-green-100 text-green-700' },
  closed: { label: 'Fechado', cls: 'bg-slate-200 text-slate-600' },
}

export const priorityLabels: Record<TicketPriority, LabelStyle> = {
  low: { label: 'Baixa', cls: 'bg-slate-100 text-slate-600' },
  medium: { label: 'Média', cls: 'bg-sky-100 text-sky-700' },
  high: { label: 'Alta', cls: 'bg-orange-100 text-orange-700' },
  critical: { label: 'Crítica', cls: 'bg-red-100 text-red-700' },
}

export const roleLabels: Record<Role, string> = {
  admin: 'Administrador',
  agent: 'Atendente',
  customer: 'Cliente',
}

/** Ordem canônica para selects de status/prioridade. */
export const statusOrder: TicketStatus[] = ['open', 'in_progress', 'waiting_customer', 'resolved', 'closed']
export const priorityOrder: TicketPriority[] = ['low', 'medium', 'high', 'critical']
