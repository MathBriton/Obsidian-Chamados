import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { ApiError } from '../lib/api'
import { AuthCard, ErrorBanner } from '../components/AuthCard'
import { TextField } from '../components/TextField'

export function RegisterPage() {
  const { register, user } = useAuth()
  const navigate = useNavigate()
  const [tenantName, setTenantName] = useState('')
  const [slug, setSlug] = useState('')
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  if (user) return <Navigate to="/" replace />

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await register({ tenant_name: tenantName, slug, name, email, password })
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao registrar')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthCard title="Criar empresa" subtitle="Cadastre sua organização e o usuário administrador">
      {error && <ErrorBanner message={error} />}
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField label="Nome da empresa" value={tenantName} onChange={setTenantName} required />
        <TextField label="Identificador (slug)" value={slug} onChange={setSlug} required placeholder="acme" />
        <TextField label="Seu nome" value={name} onChange={setName} required autoComplete="name" />
        <TextField label="E-mail" type="email" value={email} onChange={setEmail} required autoComplete="email" />
        <TextField label="Senha" type="password" value={password} onChange={setPassword} required autoComplete="new-password" placeholder="mínimo 8 caracteres" />
        <button
          type="submit"
          disabled={submitting}
          className="mt-2 rounded-lg bg-violet-600 px-4 py-2 font-medium text-white transition hover:bg-violet-700 disabled:opacity-60"
        >
          {submitting ? 'Criando…' : 'Criar empresa'}
        </button>
      </form>
      <p className="mt-6 text-center text-sm text-slate-500">
        Já tem conta?{' '}
        <Link to="/login" className="font-medium text-violet-600 hover:underline">
          Entrar
        </Link>
      </p>
    </AuthCard>
  )
}
