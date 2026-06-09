import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthProvider'
import { UsersPage } from './UsersPage'

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function seedSession(role: string) {
  localStorage.setItem(
    'obsidian.auth',
    JSON.stringify({
      accessToken: 'a',
      refreshToken: 'r',
      user: { id: 1, name: 'Admin', email: 'a@a.com', role },
      tenant: { id: 1, name: 'Acme', slug: 'acme' },
    }),
  )
}

function renderUsers() {
  return render(
    <MemoryRouter initialEntries={['/users']}>
      <AuthProvider>
        <UsersPage />
      </AuthProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('UsersPage', () => {
  it('lista os usuários do tenant com papel e status', async () => {
    seedSession('admin')
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      if (String(input).includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      return jsonResponse(200, {
        users: [
          { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin', is_active: true, created_at: '2026-01-01T00:00:00Z' },
          { id: 2, name: 'Bia', email: 'bia@a.com', role: 'agent', is_active: false, created_at: '2026-01-02T00:00:00Z' },
        ],
      })
    })

    renderUsers()

    expect(await screen.findByText('Bia')).toBeInTheDocument()
    expect(screen.getByText('bia@a.com')).toBeInTheDocument()
    // Bia é agent inativo; o rótulo "(inativo)" é exclusivo da listagem.
    expect(screen.getByText('(inativo)')).toBeInTheDocument()
  })

  it('redireciona não-admin para a home', () => {
    seedSession('customer')
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(jsonResponse(200, { users: [] }))

    renderUsers()

    // Sem heading "Usuários": foi redirecionado.
    expect(screen.queryByRole('heading', { name: 'Usuários' })).not.toBeInTheDocument()
  })
})
