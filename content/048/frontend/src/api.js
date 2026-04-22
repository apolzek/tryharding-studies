const API = '/api/v1'

async function json(path, init) {
  const r = await fetch(API + path, {
    headers: { 'content-type': 'application/json' },
    ...init,
  })
  if (!r.ok) {
    const body = await r.text()
    throw new Error(body || `HTTP ${r.status}`)
  }
  return r.json()
}

export const api = {
  register: (email, password) => json('/register', { method: 'POST', body: JSON.stringify({ email, password }) }),
  login:    (email, password) => json('/login',    { method: 'POST', body: JSON.stringify({ email, password }) }),
  tenant:   (id)              => json('/tenants/' + id),
}
