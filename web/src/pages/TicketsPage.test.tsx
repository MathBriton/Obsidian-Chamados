import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
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
    expect(screen.getByText('Aberto')).toBeInTheDocument()
    expect(screen.getByText('Alta')).toBeInTheDocument()
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
})
