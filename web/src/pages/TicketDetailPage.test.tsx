import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthProvider'
import { TicketDetailPage } from './TicketDetailPage'

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

const ticket = {
  id: 1,
  title: 'Sem rede',
  description: 'O setor está sem internet.',
  status: 'resolved',
  priority: 'high',
  category_id: 1,
  created_by: 2,
  assigned_to: 1,
  assigned_team_id: null,
  created_at: '2026-06-01T12:00:00Z',
  updated_at: '2026-06-02T12:00:00Z',
  resolved_at: '2026-06-02T12:00:00Z',
  closed_at: null,
}

const events = [
  { id: 1, kind: 'created', old_value: null, new_value: null, actor_id: 2, actor_name: 'Caio', created_at: '2026-06-01T12:00:00Z' },
  { id: 2, kind: 'assignee_changed', old_value: null, new_value: 'Ana', actor_id: 1, actor_name: 'Ana', created_at: '2026-06-01T13:00:00Z' },
  { id: 3, kind: 'status_changed', old_value: 'open', new_value: 'resolved', actor_id: 1, actor_name: 'Ana', created_at: '2026-06-02T12:00:00Z' },
]

function mockFetch(role = 'agent') {
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = String(input)
    if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Ana', email: 'a@a.com', role } })
    if (url.includes('/events')) return jsonResponse(200, { events })
    if (url.includes('/comments')) return jsonResponse(200, { comments: [] })
    if (url.includes('/assignees')) return jsonResponse(200, { users: [] })
    if (url.includes('/teams')) return jsonResponse(200, { teams: [] })
    return jsonResponse(200, ticket)
  })
}

function renderDetail() {
  return render(
    <MemoryRouter initialEntries={['/tickets/1']}>
      <AuthProvider>
        <Routes>
          <Route path="/tickets/:id" element={<TicketDetailPage />} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('TicketDetailPage — histórico', () => {
  it('mostra a timeline com os eventos traduzidos', async () => {
    seedSession('agent')
    mockFetch()

    renderDetail()

    expect(await screen.findByRole('heading', { name: 'Histórico' })).toBeInTheDocument()
    expect(screen.getByText('Chamado aberto')).toBeInTheDocument()
    expect(screen.getByText('Responsável: — → Ana')).toBeInTheDocument()
    // Valores de status são traduzidos pelos labels.
    expect(screen.getByText('Status: Aberto → Resolvido')).toBeInTheDocument()
    // Quem executou aparece pelo nome, não por "Usuário #id".
    expect(screen.getByText(/Caio ·/)).toBeInTheDocument()
  })

  it('omite a seção quando não há eventos', async () => {
    seedSession('agent')
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Ana', email: 'a@a.com', role: 'agent' } })
      if (url.includes('/events')) return jsonResponse(200, { events: [] })
      if (url.includes('/comments')) return jsonResponse(200, { comments: [] })
      if (url.includes('/assignees')) return jsonResponse(200, { users: [] })
      if (url.includes('/teams')) return jsonResponse(200, { teams: [] })
      return jsonResponse(200, ticket)
    })

    renderDetail()

    expect(await screen.findByText('Sem rede')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Histórico' })).not.toBeInTheDocument()
  })
})
