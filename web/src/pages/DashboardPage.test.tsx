import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthProvider'
import { DashboardPage } from './DashboardPage'

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function seedSession(role: string) {
  localStorage.setItem(
    'obsidian.auth',
    JSON.stringify({
      accessToken: 'a',
      refreshToken: 'r',
      user: { id: 1, name: 'Ana', email: 'a@a.com', role },
      tenant: { id: 1, name: 'Acme', slug: 'acme' },
    }),
  )
}

const stats = {
  total: 6,
  unassigned_active: 2,
  by_status: { open: 3, in_progress: 1, waiting_customer: 0, resolved: 1, closed: 1 },
  by_priority: { low: 1, medium: 3, high: 1, critical: 1 },
}

function mockFetch(role = 'agent', body: unknown = stats) {
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = String(input)
    if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Ana', email: 'a@a.com', role } })
    if (url.includes('/stats')) return jsonResponse(200, body)
    return jsonResponse(404, { error: { code: 'not_found', message: 'rota inesperada' } })
  })
}

function renderDashboard() {
  return render(
    <MemoryRouter initialEntries={['/dashboard']}>
      <AuthProvider>
        <DashboardPage />
      </AuthProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('DashboardPage', () => {
  it('mostra os cards de resumo', async () => {
    seedSession('agent')
    mockFetch()

    renderDashboard()

    // total / ativos (3+1+0) / sem responsável / resolvidos+fechados (1+1)
    expect(await screen.findByText('Total de chamados')).toBeInTheDocument()
    expect(screen.getByLabelText('Total de chamados')).toHaveTextContent('6')
    expect(screen.getByLabelText('Chamados ativos')).toHaveTextContent('4')
    expect(screen.getByLabelText('Ativos sem responsável')).toHaveTextContent('2')
    expect(screen.getByLabelText('Encerrados')).toHaveTextContent('2')
  })

  it('mostra a distribuição por status e prioridade com rótulos traduzidos', async () => {
    seedSession('agent')
    mockFetch()

    renderDashboard()

    expect(await screen.findByText('Em andamento')).toBeInTheDocument()
    expect(screen.getByText('Aguardando cliente')).toBeInTheDocument()
    expect(screen.getByText('Crítica')).toBeInTheDocument()
  })

  it('exibe erro quando a API falha', async () => {
    seedSession('agent')
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Ana', email: 'a@a.com', role: 'agent' } })
      return jsonResponse(500, { error: { code: 'internal', message: 'erro interno' } })
    })

    renderDashboard()

    expect(await screen.findByText('erro interno')).toBeInTheDocument()
  })

  it('customer vê o aviso de escopo próprio', async () => {
    seedSession('customer')
    mockFetch('customer')

    renderDashboard()

    expect(await screen.findByText(/seus chamados/i)).toBeInTheDocument()
  })
})
