export default function Dashboard({ tenant }) {
  return (
    <>
      <div className="card" data-testid="dashboard">
        <h1>tenant {tenant.tenant_id}</h1>
        <span className={`status ${tenant.status || 'pending'}`}>{tenant.status || 'pending'}</span>
        {tenant.error && <div className="error">{tenant.error}</div>}
      </div>

      <div className="card">
        <h2>ingest — OTLP/HTTP</h2>
        <Endpoint label="endpoint" value={tenant.collector_url} copy />
        <Endpoint label="token" value={tenant.ingest_token} copy masked />
        <pre style={{ background:'#000', padding:16, border:'1px solid var(--line-hi)', color:'var(--ink)', overflow:'auto' }}>
{`curl -X POST ${tenant.collector_url}/v1/traces \\
  -H "authorization: Bearer <token>" \\
  -H "content-type: application/json" \\
  -d @traces.json`}
        </pre>
      </div>

      <div className="card">
        <h2>grafana</h2>
        <Endpoint label="url"      value={tenant.grafana_url} copy link />
        <Endpoint label="user"     value="admin" copy />
        <Endpoint label="password" value={tenant.grafana_password} copy masked />
      </div>
    </>
  )
}

function Endpoint({ label, value, copy, link, masked }) {
  const display = masked ? value.slice(0, 6) + '…' + value.slice(-4) : value
  return (
    <div className="endpoint">
      <div className="label">{label}</div>
      <div className="value"><code>{link ? <a href={value} target="_blank" rel="noreferrer" style={{color:'var(--ink)'}}>{display}</a> : display}</code></div>
      {copy && <button className="copy" onClick={() => navigator.clipboard?.writeText(value)}>copy</button>}
    </div>
  )
}
