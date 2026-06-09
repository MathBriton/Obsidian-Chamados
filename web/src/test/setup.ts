// Registra os matchers do jest-dom no expect do Vitest (toBeInTheDocument, etc.)
// e limpa o DOM entre os testes.
import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'

afterEach(() => {
  cleanup()
})
