import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/context'
import { ApiError } from '../lib/api'
import { AuthCard, ErrorBanner } from '../components/AuthCard'
import { TextField } from '../components/TextField'

export function LoginPage() {
  const { login, user } = useAuth()
  const navigate = useNavigate()
  const [slug, setSlug] = useState('')
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
      await login(slug, email, password)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'falha ao entrar')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthCard title="Entrar" subtitle="Acesse o painel de chamados do seu time">
      {error && <ErrorBanner message={error} />}
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField label="Empresa (slug)" value={slug} onChange={setSlug} required autoComplete="organization" placeholder="acme" />
        <TextField label="E-mail" type="email" value={email} onChange={setEmail} required autoComplete="email" />
        <TextField label="Senha" type="password" value={password} onChange={setPassword} required autoComplete="current-password" />
        <button
          type="submit"
          disabled={submitting}
          className="mt-2 rounded-lg bg-violet-600 px-4 py-2 font-medium text-white transition hover:bg-violet-700 disabled:opacity-60"
        >
          {submitting ? 'Entrando…' : 'Entrar'}
        </button>
      </form>
      <p className="mt-6 text-center text-sm text-slate-500">
        Não tem conta?{' '}
        <Link to="/register" className="font-medium text-violet-600 hover:underline">
          Criar empresa
        </Link>
      </p>
    </AuthCard>
  )
}
