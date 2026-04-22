import { useState } from 'react'
import { api } from '../api'

export default function SignupCard({ onReady }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [mode, setMode] = useState('register')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  async function submit(e) {
    e.preventDefault()
    setBusy(true); setErr('')
    try {
      const r = mode === 'register' ? await api.register(email, password) : await api.login(email, password)
      onReady({ ...r, status: r.status || 'provisioning' })
    } catch (e) {
      setErr(e.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="card" data-testid="signup">
      <h1>{mode === 'register' ? '> provision tenant' : '> log in'}</h1>
      <p style={{ color: 'var(--ink-dim)' }}>
        {mode === 'register'
          ? 'One namespace per tenant. Collector, VictoriaMetrics, Jaeger, ClickHouse and Grafana spin up in ~60s.'
          : 'Resume your tenant dashboard.'}
      </p>
      <form onSubmit={submit}>
        <label htmlFor="email">email</label>
        <input id="email" type="email" value={email} required
               onChange={e => setEmail(e.target.value)} />
        <label htmlFor="pw">password (12+ chars)</label>
        <input id="pw" type="password" value={password} required minLength={12}
               onChange={e => setPassword(e.target.value)} />
        {err && <div className="error" role="alert">{err}</div>}
        <button type="submit" disabled={busy}>
          {busy ? 'working…' : (mode === 'register' ? 'provision' : 'log in')}
        </button>
      </form>
      <p style={{ marginTop: 28, color: 'var(--ink-fade)' }}>
        {mode === 'register'
          ? <>Already have a tenant? <a href="#" onClick={(e)=>{e.preventDefault(); setMode('login')}} style={{color:'var(--ink)'}}>log in</a></>
          : <>Need a tenant? <a href="#" onClick={(e)=>{e.preventDefault(); setMode('register')}} style={{color:'var(--ink)'}}>sign up</a></>}
      </p>
    </div>
  )
}
