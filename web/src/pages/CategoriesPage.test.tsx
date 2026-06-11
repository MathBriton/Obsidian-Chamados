import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthProvider'
import { CategoriesPage } from './CategoriesPage'

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

function renderCategories() {
  return render(
    <MemoryRouter initialEntries={['/categories']}>
      <AuthProvider>
        <CategoriesPage />
      </AuthProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('CategoriesPage', () => {
  it('lista as categorias do tenant', async () => {
    seedSession('admin')
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      if (String(input).includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      return jsonResponse(200, {
        categories: [
          { id: 1, name: 'Infraestrutura' },
          { id: 2, name: 'Financeiro' },
        ],
      })
    })

    renderCategories()

    expect(await screen.findByText('Infraestrutura')).toBeInTheDocument()
    expect(screen.getByText('Financeiro')).toBeInTheDocument()
  })

  it('mostra estado vazio quando não há categorias', async () => {
    seedSession('admin')
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      if (String(input).includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      return jsonResponse(200, { categories: [] })
    })

    renderCategories()

    expect(await screen.findByText(/nenhuma categoria/i)).toBeInTheDocument()
  })

  it('cria uma categoria e recarrega a lista', async () => {
    seedSession('admin')
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      if (url.includes('/categories') && init?.method === 'POST') {
        return jsonResponse(201, { id: 3, name: 'Suporte' })
      }
      return jsonResponse(200, { categories: [] })
    })

    renderCategories()
    await screen.findByText(/nenhuma categoria/i)

    await userEvent.type(screen.getByLabelText('Nome'), 'Suporte')
    await userEvent.click(screen.getByRole('button', { name: 'Criar categoria' }))

    const postCall = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')
    expect(postCall).toBeDefined()
    expect(JSON.parse(String(postCall?.[1]?.body))).toEqual({ name: 'Suporte' })
  })

  it('exibe o erro da API ao falhar a criação', async () => {
    seedSession('admin')
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input, init) => {
      const url = String(input)
      if (url.includes('/me')) return jsonResponse(200, { user: { id: 1, name: 'Admin', email: 'a@a.com', role: 'admin' } })
      if (url.includes('/categories') && init?.method === 'POST') {
        return jsonResponse(409, { error: { code: 'category_exists', message: 'categoria já existe' } })
      }
      return jsonResponse(200, { categories: [{ id: 1, name: 'Suporte' }] })
    })

    renderCategories()
    await screen.findByText('Suporte')

    await userEvent.type(screen.getByLabelText('Nome'), 'Suporte')
    await userEvent.click(screen.getByRole('button', { name: 'Criar categoria' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('categoria já existe')
  })

  it('redireciona não-admin para a home', () => {
    seedSession('customer')
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(jsonResponse(200, { categories: [] }))

    renderCategories()

    // Sem heading "Categorias": foi redirecionado.
    expect(screen.queryByRole('heading', { name: 'Categorias' })).not.toBeInTheDocument()
  })
})
