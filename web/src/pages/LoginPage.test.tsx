import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthProvider'
import { LoginPage } from './LoginPage'

function renderLogin() {
  return render(
    <MemoryRouter initialEntries={['/login']}>
      <AuthProvider>
        <LoginPage />
      </AuthProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('LoginPage', () => {
  it('renderiza os campos de login', () => {
    renderLogin()
    expect(screen.getByLabelText('Empresa (slug)')).toBeInTheDocument()
    expect(screen.getByLabelText('E-mail')).toBeInTheDocument()
    expect(screen.getByLabelText('Senha')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Entrar' })).toBeInTheDocument()
  })

  it('mostra a mensagem de erro quando o login falha', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'invalid_credentials', message: 'credenciais inválidas' } }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    const user = userEvent.setup()
    renderLogin()

    await user.type(screen.getByLabelText('Empresa (slug)'), 'acme')
    await user.type(screen.getByLabelText('E-mail'), 'admin@acme.com')
    await user.type(screen.getByLabelText('Senha'), 'senha-errada')
    await user.click(screen.getByRole('button', { name: 'Entrar' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('credenciais inválidas')
  })

  it('envia as credenciais digitadas para a API', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'invalid_credentials', message: 'x' } }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    const user = userEvent.setup()
    renderLogin()

    await user.type(screen.getByLabelText('Empresa (slug)'), 'acme')
    await user.type(screen.getByLabelText('E-mail'), 'admin@acme.com')
    await user.type(screen.getByLabelText('Senha'), 'segredo123')
    await user.click(screen.getByRole('button', { name: 'Entrar' }))

    await screen.findByRole('alert')
    const [url, init] = fetchSpy.mock.calls[0]
    expect(String(url)).toContain('/auth/login')
    expect(JSON.parse(init?.body as string)).toEqual({
      slug: 'acme',
      email: 'admin@acme.com',
      password: 'segredo123',
    })
  })
})
