import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../api.js';

export default function Politicians() {
  const { sigla } = useParams();
  if (sigla) return <PartyDetail sigla={sigla} />;
  return <Overview />;
}

function Overview() {
  const navigate = useNavigate();
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [filter, setFilter] = useState('');

  useEffect(() => {
    api.partidos().then(setData).catch((e) => setError(e.message));
  }, []);

  const filtered = useMemo(() => {
    if (!data) return [];
    const q = filter.trim().toLowerCase();
    if (!q) return data.partidos;
    return data.partidos.filter((p) => p.partido.toLowerCase().includes(q));
  }, [data, filter]);

  if (error) return <div className="error">Erro ao carregar: {error} <br/> <small>As APIs da Câmara/Senado podem estar temporariamente indisponíveis.</small></div>;
  if (!data) return <div className="loading">Consultando Câmara e Senado em tempo real…</div>;

  const max = Math.max(...data.partidos.map((p) => p.total), 1);

  return (
    <div>
      <h2>Políticos em exercício</h2>
      <p style={{ color: 'var(--muted)' }}>Dados em tempo real das APIs oficiais da Câmara dos Deputados e do Senado Federal.</p>

      <div className="stats">
        <div className="stat"><span className="n">{data.totals.deputados}</span><span className="lbl">Deputados</span></div>
        <div className="stat"><span className="n">{data.totals.senadores}</span><span className="lbl">Senadores</span></div>
        <div className="stat"><span className="n">{data.totals.partidos}</span><span className="lbl">Partidos</span></div>
        <div className="stat"><span className="n">{data.totals.total}</span><span className="lbl">Total</span></div>
      </div>

      <div className="card">
        <h3>Distribuição por partido (Câmara + Senado)</h3>
        <input
          type="text"
          className="input"
          placeholder="Filtrar sigla..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
        <div className="bar-chart" style={{ marginTop: '1rem' }}>
          {filtered.map((p) => (
            <div key={p.partido} className="bar-row" onClick={() => navigate(`/politicos/${p.partido}`)} style={{ cursor: 'pointer' }}>
              <div className="label">{p.partido}</div>
              <div className="bar" style={{ width: `${(p.total / max) * 100}%` }} />
              <div className="count">{p.total}</div>
            </div>
          ))}
        </div>
        <small style={{ color: 'var(--muted)' }}>Clique em uma barra para ver detalhes do partido.</small>
      </div>
    </div>
  );
}

function PartyDetail({ sigla }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    setData(null);
    setError(null);
    api.partido(sigla).then(setData).catch((e) => setError(e.message));
  }, [sigla]);

  if (error) return <div className="error">Erro: {error}</div>;
  if (!data) return <div className="loading">Carregando partido {sigla}…</div>;

  const ufs = Object.entries(data.porUf).sort((a, b) => b[1] - a[1]);
  const maxUf = Math.max(...ufs.map(([, n]) => n), 1);

  return (
    <div>
      <div className="breadcrumb"><Link to="/politicos">← Políticos</Link> / {sigla}</div>
      <div className="card" style={{ borderTopColor: 'var(--azul)' }}>
        <h2 style={{ color: 'var(--azul)' }}>{sigla}</h2>
        <div className="stats">
          <div className="stat"><span className="n">{data.total}</span><span className="lbl">Parlamentares</span></div>
          <div className="stat"><span className="n">{data.camara}</span><span className="lbl">Câmara</span></div>
          <div className="stat"><span className="n">{data.senado}</span><span className="lbl">Senado</span></div>
          <div className="stat"><span className="n">{ufs.length}</span><span className="lbl">UFs</span></div>
        </div>

        <h3>Distribuição por UF</h3>
        <div className="bar-chart">
          {ufs.map(([uf, n]) => (
            <div key={uf} className="bar-row">
              <div className="label">{uf}</div>
              <div className="bar" style={{ width: `${(n / maxUf) * 100}%`, background: 'var(--azul)' }} />
              <div className="count">{n}</div>
            </div>
          ))}
        </div>
      </div>

      {data.senadores.length > 0 && (
        <div className="card">
          <h3>Senadores ({data.senadores.length})</h3>
          <div className="politician-grid">
            {data.senadores.map((p) => (
              <div key={p.id} className="politician">
                {p.foto && <img src={p.foto} alt={p.nome} onError={(e) => (e.currentTarget.style.display = 'none')} />}
                <div className="name">{p.nome}</div>
                <div style={{ color: 'var(--muted)' }}>{p.uf}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {data.deputados.length > 0 && (
        <div className="card">
          <h3>Deputados ({data.deputados.length})</h3>
          <div className="politician-grid">
            {data.deputados.map((p) => (
              <div key={p.id} className="politician">
                {p.foto && <img src={p.foto} alt={p.nome} onError={(e) => (e.currentTarget.style.display = 'none')} />}
                <div className="name">{p.nome}</div>
                <div style={{ color: 'var(--muted)' }}>{p.uf}</div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
