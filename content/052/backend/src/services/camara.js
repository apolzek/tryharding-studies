import { withCache } from './cache.js';

const CAMARA_BASE = 'https://dadosabertos.camara.leg.br/api/v2';
const SENADO_BASE = 'https://legis.senado.leg.br/dadosabertos';
const TTL = 1000 * 60 * 60 * 6;

async function fetchJson(url, headers = {}) {
  const res = await fetch(url, { headers: { Accept: 'application/json', ...headers } });
  if (!res.ok) throw new Error(`upstream ${res.status} for ${url}`);
  return res.json();
}

export async function listDeputados() {
  return withCache('deputados', TTL, async () => {
    const all = [];
    let url = `${CAMARA_BASE}/deputados?ordem=ASC&ordenarPor=nome&itens=100`;
    while (url) {
      const data = await fetchJson(url);
      all.push(...(data.dados || []));
      const next = (data.links || []).find((l) => l.rel === 'next');
      url = next ? next.href : null;
      if (all.length > 700) break;
    }
    return all.map((d) => ({
      id: d.id,
      nome: d.nome,
      partido: d.siglaPartido,
      uf: d.siglaUf,
      foto: d.urlFoto,
      email: d.email,
      legislatura: d.idLegislatura,
    }));
  });
}

export async function listSenadores() {
  return withCache('senadores', TTL, async () => {
    const data = await fetchJson(`${SENADO_BASE}/senador/lista/atual`);
    const list = data?.ListaParlamentarEmExercicio?.Parlamentares?.Parlamentar || [];
    return list.map((s) => {
      const i = s.IdentificacaoParlamentar || {};
      const m = s.Mandato || {};
      return {
        id: i.CodigoParlamentar,
        nome: i.NomeParlamentar,
        nomeCivil: i.NomeCompletoParlamentar,
        partido: i.SiglaPartidoParlamentar,
        uf: i.UfParlamentar,
        foto: i.UrlFotoParlamentar,
        email: i.EmailParlamentar,
        legislatura: m.PrimeiraLegislaturaDoMandato?.NumeroLegislatura,
      };
    });
  });
}

export function aggregateByParty(people) {
  const map = new Map();
  for (const p of people) {
    if (!p.partido) continue;
    const key = p.partido;
    if (!map.has(key)) map.set(key, { partido: key, total: 0, porUf: {}, membros: [] });
    const row = map.get(key);
    row.total += 1;
    row.porUf[p.uf] = (row.porUf[p.uf] || 0) + 1;
    row.membros.push({ id: p.id, nome: p.nome, uf: p.uf, foto: p.foto });
  }
  return [...map.values()].sort((a, b) => b.total - a.total);
}
