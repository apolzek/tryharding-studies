import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import Dashboard from './Dashboard.jsx'

const t = {
  tenant_id: 't-abc',
  ingest_token: 'tokABCDEFGHJKLMNOP1234567890', // pragma: allowlist secret
  grafana_password: 'mysecret1234', // pragma: allowlist secret
  collector_url: 'http://t-abc-ingest.localtest.me',
  grafana_url: 'http://t-abc-grafana.localtest.me',
  status: 'ready',
}

describe('Dashboard', () => {
  it('renders tenant id and status', () => {
    render(<Dashboard tenant={t} />)
    expect(screen.getByText(/tenant t-abc/i)).toBeInTheDocument()
    expect(screen.getByText('ready')).toBeInTheDocument()
  })

  it('masks the ingest token', () => {
    render(<Dashboard tenant={t} />)
    // Token is masked — full token text should NOT be visible.
    expect(screen.queryByText(t.ingest_token)).toBeNull()
  })

  it('shows collector endpoint', () => {
    render(<Dashboard tenant={t} />)
    expect(screen.getByText(t.collector_url)).toBeInTheDocument()
  })
})
