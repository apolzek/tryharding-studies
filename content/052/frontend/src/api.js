const BASE = '/api';

async function json(path) {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new Error(`API ${res.status}: ${body || path}`);
  }
  return res.json();
}

export const api = {
  tipToday: () => json('/tips/today'),
  tipsAll: () => json('/tips'),
  rightsList: () => json('/rights'),
  rightsDetail: (id) => json(`/rights/${id}`),
  taxes: () => json('/taxes'),
  history: () => json('/history'),
  deputados: () => json('/politicians/deputados'),
  senadores: () => json('/politicians/senadores'),
  partidos: (scope) => json(`/politicians/partidos${scope ? `?scope=${scope}` : ''}`),
  partido: (sigla) => json(`/politicians/partidos/${encodeURIComponent(sigla)}`),
};
