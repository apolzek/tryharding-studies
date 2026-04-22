import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { vi, describe, it, expect, beforeEach } from 'vitest'
import SignupCard from './SignupCard.jsx'

const mockFetch = (impl) => { global.fetch = vi.fn(impl) }

describe('SignupCard', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('submits register and calls onReady with tenant', async () => {
    mockFetch(() => Promise.resolve({
      ok: true,
      json: () => Promise.resolve({
        tenant_id: 't-abc',
        ingest_token: 'tok',
        grafana_password: 'pwd', // pragma: allowlist secret
        collector_url: 'http://t-abc-ingest.localtest.me',
        grafana_url: 'http://t-abc-grafana.localtest.me',
      }),
    }))
    const onReady = vi.fn()
    render(<SignupCard onReady={onReady} />)
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'a@b.com' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'pw-pw-pw-pw-pw' } })
    fireEvent.click(screen.getByRole('button', { name: /provision/i }))
    await waitFor(() => expect(onReady).toHaveBeenCalled())
    expect(onReady.mock.calls[0][0].tenant_id).toBe('t-abc')
  })

  it('shows server error', async () => {
    mockFetch(() => Promise.resolve({ ok: false, status: 409, text: () => Promise.resolve('email already registered') }))
    render(<SignupCard onReady={() => {}} />)
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'a@b.com' } })
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: 'pw-pw-pw-pw-pw' } })
    fireEvent.click(screen.getByRole('button', { name: /provision/i }))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(/already registered/i))
  })

  it('switches to login mode', () => {
    render(<SignupCard onReady={() => {}} />)
    fireEvent.click(screen.getByText(/log in/i))
    expect(screen.getByRole('button', { name: /log in/i })).toBeInTheDocument()
  })
})
