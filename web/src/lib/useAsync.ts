import { useCallback, useEffect, useState } from 'react'
import { ApiError } from './api'

/** Executa um loader assíncrono memoizado, expondo data/error/loading e um
 * reload. O `loader` deve ser estável (useCallback) para evitar re-execuções. */
export function useAsync<T>(loader: () => Promise<T>) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const run = useCallback(() => {
    let active = true
    setLoading(true)
    setError(null)
    loader()
      .then((d) => {
        if (active) setData(d)
      })
      .catch((e) => {
        if (active) setError(e instanceof ApiError ? e.message : 'erro inesperado')
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [loader])

  // Busca ao montar (e quando o loader muda). O setState dentro do efeito é
  // intencional aqui — é o padrão de data-fetching com estado de carregamento.
  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => run(), [run])

  return { data, error, loading, reload: run, setData }
}
