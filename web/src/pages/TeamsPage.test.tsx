import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthProvider'
import { TeamsPage } from './TeamsPage'

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

const teams = [
  {
    id: 1,
    name: 'Infraestrutura',
    members: [{ id: 2, name: 'Bia', role: 'agent' }],
  },
  { id: 2, name: 'Suporte N1', members: [] },
]

const assignees = [
  { id: 1, name: 'Admin', role: 'admin' },
  { id: 2, name: 'Bia', role: 'agent' },
  { id: 3, name: 'Caio', role: 'agent' },
]

function mockFetch() {
  return vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
    const url = String(input)
    if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
    if (url.includes('/assignees')) return jsonResponse(200, { users: assignees })
    if (url.includes('/members') && init?.method === 'POST') return new Response(null, { status: 204 })
    if (url.includes('/members') && init?.method === 'DELETE') return new Response(null, { status: 204 })
    if (url.includes('/teams') && init?.method === 'POST') return jsonResponse(201, { id: 3, name: 'Nova', members: [] })
    return jsonResponse(200, { teams })
  })
}

function renderTeams() {
  return render(
    <MemoryRouter initialEntries={['/teams']}>
      <AuthProvider>
        <TeamsPage />
      </AuthProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('TeamsPage', () => {
  it('lista as equipes com seus membros', async () => {
    seedSession('admin')
    mockFetch()

    renderTeams()

    const infraCard = (await screen.findByText('Infraestrutura')).closest('li') as HTMLElement
    const suporteCard = screen.getByText('Suporte N1').closest('li') as HTMLElement

    // Bia é membro da Infraestrutura; Suporte N1 ainda não tem membros.
    expect(within(infraCard).getByText('Bia')).toBeInTheDocument()
    expect(within(suporteCard).getByText('Nenhum membro ainda.')).toBeInTheDocument()
  })

  it('cria uma equipe', async () => {
    seedSession('admin')
    const fetchMock = mockFetch()

    renderTeams()
    await screen.findByText('Infraestrutura')

    await userEvent.type(screen.getByLabelText('Nome'), 'Nova')
    await userEvent.click(screen.getByRole('button', { name: 'Criar equipe' }))

    const post = fetchMock.mock.calls.find(([u, init]) => String(u).endsWith('/teams') && init?.method === 'POST')
    expect(post).toBeDefined()
    expect(JSON.parse(String(post?.[1]?.body))).toEqual({ name: 'Nova' })
  })

  it('adiciona um membro à equipe', async () => {
    seedSession('admin')
    const fetchMock = mockFetch()

    renderTeams()
    const teamCard = (await screen.findByText('Infraestrutura')).closest('li') as HTMLElement

    // Caio (id 3) ainda não é membro da Infraestrutura.
    await userEvent.selectOptions(within(teamCard).getByLabelText('Adicionar membro'), '3')

    const post = fetchMock.mock.calls.find(
      ([u, init]) => String(u).includes('/teams/1/members') && init?.method === 'POST',
    )
    expect(post).toBeDefined()
    expect(JSON.parse(String(post?.[1]?.body))).toEqual({ user_id: 3 })
  })

  it('remove um membro da equipe', async () => {
    seedSession('admin')
    const fetchMock = mockFetch()

    renderTeams()
    const teamCard = (await screen.findByText('Infraestrutura')).closest('li') as HTMLElement

    await userEvent.click(within(teamCard).getByRole('button', { name: 'Remover Bia' }))

    expect(
      fetchMock.mock.calls.some(([u, init]) => String(u).includes('/teams/1/members/2') && init?.method === 'DELETE'),
    ).toBe(true)
  })

  it('redireciona não-admin para a home', () => {
    seedSession('agent')
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(jsonResponse(200, { teams: [] }))

    renderTeams()

    expect(screen.queryByRole('heading', { name: 'Equipes' })).not.toBeInTheDocument()
  })
})
