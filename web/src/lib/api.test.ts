import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api } from './api'

afterEach(() => {
  vi.restoreAllMocks()
})

function mockFetch(status: number, body: unknown) {
  return vi.spyOn(globalThis, 'fetch').mockResolvedValue(
    new Response(body === undefined ? null : JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
}

describe('api client', () => {
  it('faz login e devolve o resultado tipado', async () => {
    const result = {
      access_token: 'a',
      refresh_token: 'r',
      user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' },
      tenant: { id: 1, name: 'Acme', slug: 'acme' },
    }
    const fetchSpy = mockFetch(200, result)

    await expect(api.login({ slug: 'acme', email: 'a@a.com', password: 'segredo123' })).resolves.toEqual(result)

    const [, init] = fetchSpy.mock.calls[0]
    expect(init?.method).toBe('POST')
    expect(JSON.parse(init?.body as string)).toEqual({ slug: 'acme', email: 'a@a.com', password: 'segredo123' })
  })

  it('lança ApiError com o código estável do backend', async () => {
    mockFetch(401, { error: { code: 'invalid_credentials', message: 'credenciais inválidas' } })

    await expect(api.login({ slug: 'acme', email: 'a@a.com', password: 'errada' })).rejects.toMatchObject({
      status: 401,
      code: 'invalid_credentials',
    })
  })

  it('envia o Bearer token em rotas autenticadas', async () => {
    const fetchSpy = mockFetch(200, { user: { id: 1, name: 'A', email: 'a@a.com', role: 'admin' } })

    await api.me('token-123')

    const [, init] = fetchSpy.mock.calls[0]
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer token-123')
  })

  it('traduz falha de rede em ApiError network_error', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('failed'))

    await expect(api.me('x')).rejects.toBeInstanceOf(ApiError)
    await expect(api.me('x')).rejects.toMatchObject({ code: 'network_error' })
  })

  it('lista tickets desembrulhando o array da resposta', async () => {
    const tickets = [{ id: 1, title: 'Falha', status: 'open' }]
    mockFetch(200, { tickets })

    await expect(api.listTickets('tok')).resolves.toEqual(tickets)
  })

  it('serializa filtros de tickets como query string, omitindo vazios', async () => {
    const fetchSpy = mockFetch(200, { tickets: [] })

    await api.listTickets('tok', { status: 'open', priority: 'high', q: 'login', assigned_to: undefined })

    const url = String(fetchSpy.mock.calls[0][0])
    expect(url).toContain('/tickets?')
    expect(url).toContain('status=open')
    expect(url).toContain('priority=high')
    expect(url).toContain('q=login')
    expect(url).not.toContain('assigned_to')
  })

  it('lista tickets sem query string quando não há filtros', async () => {
    const fetchSpy = mockFetch(200, { tickets: [] })

    await api.listTickets('tok')

    expect(String(fetchSpy.mock.calls[0][0])).not.toContain('?')
  })

  it('cria ticket com método POST e Bearer token', async () => {
    const fetchSpy = mockFetch(201, { id: 9, title: 'Novo' })

    await api.createTicket('tok', { title: 'Novo', description: 'desc', category_id: 2, priority: 'high' })

    const [url, init] = fetchSpy.mock.calls[0]
    expect(String(url)).toContain('/tickets')
    expect(init?.method).toBe('POST')
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer tok')
    expect(JSON.parse(init?.body as string)).toEqual({
      title: 'Novo',
      description: 'desc',
      category_id: 2,
      priority: 'high',
    })
  })

  it('lista responsáveis atribuíveis desembrulhando o array', async () => {
    const users = [{ id: 2, name: 'Ana', role: 'agent' }]
    const fetchSpy = mockFetch(200, { users })

    await expect(api.listAssignees('tok')).resolves.toEqual(users)
    expect(String(fetchSpy.mock.calls[0][0])).toContain('/assignees')
  })

  it('adiciona comentário no endpoint do ticket', async () => {
    const fetchSpy = mockFetch(201, { id: 1, ticket_id: 5, body: 'oi', is_internal: false })

    await api.addComment('tok', 5, 'oi', true)

    const [url, init] = fetchSpy.mock.calls[0]
    expect(String(url)).toContain('/tickets/5/comments')
    expect(JSON.parse(init?.body as string)).toEqual({ body: 'oi', is_internal: true })
  })
})
