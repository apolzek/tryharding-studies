import { useEffect, useState } from 'react';
import { api } from '../api.js';

export default function History() {
  const [data, setData] = useState(null);
  const [open, setOpen] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    api.history().then((d) => {
      setData(d);
      setOpen(d.periods[0]?.id);
    }).catch((e) => setError(e.message));
  }, []);

  if (error) return <div className="error">Erro: {error}</div>;
  if (!data) return <div className="loading">Carregando…</div>;

  const periodo = data.periods.find((p) => p.id === open);

  return (
    <div>
      <h2>História do Brasil</h2>
      <p style={{ color: 'var(--muted)' }}>
        Mais de 500 anos em marcos resumidos. Clique em um período para ver a linha do tempo.
      </p>

      <div className="tabs">
        {data.periods.map((p) => (
          <button key={p.id} className={open === p.id ? 'active' : ''} onClick={() => setOpen(p.id)}>
            {p.range}
          </button>
        ))}
      </div>

      {periodo && (
        <div className="card">
          <h3 style={{ color: 'var(--azul)' }}>{periodo.title}</h3>
          <p>{periodo.summary}</p>
          <div className="timeline">
            {periodo.events.map((e, i) => (
              <div key={i} className="event">
                <span className="year">{e.year} — </span>
                <strong>{e.title}</strong>
                <p style={{ margin: '0.2rem 0 0', color: 'var(--muted)' }}>{e.desc}</p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
