import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthProvider'
import { TicketsPage } from './TicketsPage'

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function seedSession() {
  localStorage.setItem(
    'obsidian.auth',
    JSON.stringify({
      accessToken: 'a',
      refreshToken: 'r',
      user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' },
      tenant: { id: 1, name: 'Acme', slug: 'acme' },
    }),
  )
}

function renderTickets() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <TicketsPage />
      </AuthProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
  seedSession()
})

describe('TicketsPage', () => {
  it('lista os tickets com título e status', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      return jsonResponse(200, {
        tickets: [{ id: 1, title: 'Servidor fora do ar', status: 'open', priority: 'high' }],
      })
    })

    renderTickets()

    expect(await screen.findByText('Servidor fora do ar')).toBeInTheDocument()
    const item = screen.getByRole('listitem')
    expect(within(item).getByText('Aberto')).toBeInTheDocument()
    expect(within(item).getByText('Alta')).toBeInTheDocument()
  })

  it('mostra estado vazio quando não há tickets', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      return jsonResponse(200, { tickets: [] })
    })

    renderTickets()

    expect(await screen.findByText(/Nenhum chamado ainda/)).toBeInTheDocument()
  })

  it('refaz a busca com o filtro de status na query string', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      if (url.includes('status=resolved')) {
        return jsonResponse(200, { tickets: [{ id: 2, title: 'Já resolvido', status: 'resolved', priority: 'low' }] })
      }
      return jsonResponse(200, { tickets: [{ id: 1, title: 'Em aberto', status: 'open', priority: 'high' }] })
    })

    renderTickets()
    expect(await screen.findByText('Em aberto')).toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Filtrar por status'), 'resolved')

    expect(await screen.findByText('Já resolvido')).toBeInTheDocument()
    expect(screen.queryByText('Em aberto')).not.toBeInTheDocument()
    const filtered = fetchSpy.mock.calls.map((c) => String(c[0])).filter((u) => u.includes('status=resolved'))
    expect(filtered.length).toBeGreaterThan(0)
  })

  it('pagina com "Carregar mais": anexa a próxima página e esconde o botão no fim', async () => {
    const page1 = Array.from({ length: 20 }, (_, i) => ({
      id: i + 1,
      title: `Chamado ${i + 1}`,
      status: 'open',
      priority: 'medium',
    }))
    const page2 = [{ id: 21, title: 'Chamado 21', status: 'open', priority: 'medium' }]

    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      if (url.includes('offset=20')) return jsonResponse(200, { tickets: page2 })
      return jsonResponse(200, { tickets: page1 })
    })

    renderTickets()
    expect(await screen.findByText('Chamado 1')).toBeInTheDocument()
    expect(screen.getAllByRole('listitem')).toHaveLength(20)

    // Primeira página veio cheia: botão visível; pede a próxima com offset=20.
    await userEvent.click(screen.getByRole('button', { name: 'Carregar mais' }))

    expect(await screen.findByText('Chamado 21')).toBeInTheDocument()
    expect(screen.getAllByRole('listitem')).toHaveLength(21)
    expect(fetchSpy.mock.calls.some(([u]) => String(u).includes('limit=20') && String(u).includes('offset=20'))).toBe(true)
    // Página incompleta: acabou, botão some.
    expect(screen.queryByRole('button', { name: 'Carregar mais' })).not.toBeInTheDocument()
  })

  it('não mostra "Carregar mais" quando a primeira página vem incompleta', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      return jsonResponse(200, { tickets: [{ id: 1, title: 'Único', status: 'open', priority: 'low' }] })
    })

    renderTickets()

    expect(await screen.findByText('Único')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Carregar mais' })).not.toBeInTheDocument()
  })

  it('envia a busca por texto após o debounce e mostra vazio filtrado', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      if (url.includes('q=impressora')) return jsonResponse(200, { tickets: [] })
      return jsonResponse(200, { tickets: [{ id: 1, title: 'Em aberto', status: 'open', priority: 'high' }] })
    })

    renderTickets()
    expect(await screen.findByText('Em aberto')).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Buscar chamados'), 'impressora')

    await waitFor(() => expect(screen.getByText(/Nenhum chamado corresponde aos filtros/)).toBeInTheDocument())
  })
})
