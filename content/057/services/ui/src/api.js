const API_BASE = import.meta.env.VITE_API_BASE || '/api';

async function json(res) {
  const text = await res.text();
  try { return JSON.parse(text); } catch { return text; }
}

let authToken = localStorage.getItem('057.token') || '';

export function setToken(t) { authToken = t; localStorage.setItem('057.token', t || ''); }
export function getToken() { return authToken; }

function headers(extra = {}) {
  const h = { 'content-type': 'application/json', ...extra };
  if (authToken) h.Authorization = `Bearer ${authToken}`;
  return h;
}

export const api = {
  login: (email, password) =>
    fetch(`${API_BASE}/auth/login`, { method: 'POST', headers: headers(), body: JSON.stringify({ email, password }) }).then(json),

  register: (email, password) =>
    fetch(`${API_BASE}/auth/register`, { method: 'POST', headers: headers(), body: JSON.stringify({ email, password }) }).then(json),

  listProducts: () => fetch(`${API_BASE}/products?limit=50`, { headers: headers() }).then(json),
  createProduct: (p) => fetch(`${API_BASE}/products`, { method: 'POST', headers: headers(), body: JSON.stringify(p) }).then(json),

  listCustomers: () => fetch(`${API_BASE}/customers`, { headers: headers() }).then(json),

  summary: () => fetch(`${API_BASE}/signatures/summary`, { headers: headers() }).then(json),

  checkout: (payload, idemKey) =>
    fetch(`${API_BASE}/checkout`, {
      method: 'POST',
      headers: headers(idemKey ? { 'Idempotency-Key': idemKey } : {}),
      body: JSON.stringify(payload),
    }).then(json),

  listOrders: (customerId) =>
    fetch(`${API_BASE}/orders/by-customer/${encodeURIComponent(customerId)}`, { headers: headers() }).then(json),
  orderHistory: (id) => fetch(`${API_BASE}/orders/${id}/history`, { headers: headers() }).then(json),
  fulfill: (id) => fetch(`${API_BASE}/orders/${id}/fulfill`, { method: 'POST', headers: headers() }).then(json),
  cancel: (id) => fetch(`${API_BASE}/orders/${id}/cancel`, { method: 'POST', headers: headers() }).then(json),

  cart: {
    get: (id) => fetch(`${API_BASE}/carts/${id}`, { headers: headers() }).then(json),
    add: (id, item) => fetch(`${API_BASE}/carts/${id}/items`, { method: 'POST', headers: headers(), body: JSON.stringify(item) }).then(json),
    remove: (id, pid) => fetch(`${API_BASE}/carts/${id}/items/${pid}`, { method: 'DELETE', headers: headers() }).then(json),
    clear: (id) => fetch(`${API_BASE}/carts/${id}`, { method: 'DELETE', headers: headers() }).then(json),
  },

  notifications: () => fetch(`${API_BASE}/notifications/recent?limit=30`, { headers: headers() }).then(json),
};

export function subscribeEvents(onEvent) {
  const es = new EventSource(`${API_BASE}/events`);
  const types = ['customer.events', 'order.events', 'payment.events'];
  for (const t of types) {
    es.addEventListener(t, (e) => {
      try { onEvent({ type: t, data: JSON.parse(e.data) }); } catch {}
    });
  }
  es.onmessage = (e) => {
    try { onEvent({ type: 'message', data: JSON.parse(e.data) }); } catch {}
  };
  return es;
}
