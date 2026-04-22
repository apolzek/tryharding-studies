import { useEffect, useState } from 'react'
import { api } from './api'
import SignupCard from './components/SignupCard.jsx'
import Dashboard from './components/Dashboard.jsx'

export default function App() {
  const [tenant, setTenant] = useState(() => {
    try { return JSON.parse(sessionStorage.getItem('tenant')) } catch { return null }
  })

  useEffect(() => {
    if (tenant) sessionStorage.setItem('tenant', JSON.stringify(tenant))
    else sessionStorage.removeItem('tenant')
  }, [tenant])

  // If we have a tenant, poll its status until ready.
  useEffect(() => {
    if (!tenant || tenant.status === 'ready' || tenant.status === 'failed') return
    const t = setInterval(async () => {
      try {
        const s = await api.tenant(tenant.tenant_id)
        setTenant(prev => ({ ...prev, status: s.status, error: s.error }))
      } catch (_) { /* retry */ }
    }, 2500)
    return () => clearInterval(t)
  }, [tenant])

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">OBS/SAAS <small>— observability delivered</small></div>
        {tenant && <button className="ghost" onClick={() => setTenant(null)}>Log out</button>}
      </header>
      <main className="container">
        {tenant ? <Dashboard tenant={tenant} /> : <SignupCard onReady={setTenant} />}
      </main>
      <footer>© obs/saas — kind · helm · otel · victoria-metrics · jaeger · clickhouse · grafana</footer>
    </div>
  )
}
