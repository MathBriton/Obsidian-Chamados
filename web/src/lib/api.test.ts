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
})
