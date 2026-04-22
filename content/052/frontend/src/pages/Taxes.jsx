import { useEffect, useState } from 'react';
import { api } from '../api.js';

const ORDER = ['federal', 'estadual', 'municipal'];
const TITLES = { federal: 'Impostos Federais', estadual: 'Impostos Estaduais', municipal: 'Impostos Municipais' };

export default function Taxes() {
  const [data, setData] = useState(null);
  const [tab, setTab] = useState('federal');
  const [error, setError] = useState(null);

  useEffect(() => {
    api.taxes().then(setData).catch((e) => setError(e.message));
  }, []);

  if (error) return <div className="error">Erro: {error}</div>;
  if (!data) return <div className="loading">Carregando…</div>;

  return (
    <div>
      <h2>Impostos no Brasil</h2>
      <p style={{ color: 'var(--muted)' }}>
        O sistema tributário brasileiro é conhecido por sua complexidade. Aqui um resumo dos principais tributos nos três níveis federativos e do calendário anual.
      </p>

      <div className="tabs">
        {ORDER.map((t) => (
          <button key={t} className={tab === t ? 'active' : ''} onClick={() => setTab(t)}>
            {TITLES[t]}
          </button>
        ))}
        <button className={tab === 'calendario' ? 'active' : ''} onClick={() => setTab('calendario')}>
          Calendário
        </button>
      </div>

      {tab !== 'calendario' && (
        <div>
          {(data[tab] || []).map((item) => (
            <div key={item.id} className="card">
              <h3>{item.name}</h3>
              <p>{item.description}</p>
              {item.rate && <p><strong>Alíquota:</strong> {item.rate}</p>}
              {item.deadline && <p><strong>Prazo:</strong> {item.deadline}</p>}
              {item.tips && (
                <ul style={{ paddingLeft: '1.2rem' }}>
                  {item.tips.map((t, i) => <li key={i}>{t}</li>)}
                </ul>
              )}
              {item.link && <a href={item.link} target="_blank" rel="noreferrer">site oficial →</a>}
            </div>
          ))}
        </div>
      )}

      {tab === 'calendario' && (
        <div className="card">
          <h3>Calendário Tributário</h3>
          {data.calendario.map((c, i) => (
            <div key={i} className="item">
              <h4>{c.month}</h4>
              <ul>
                {c.events.map((e, j) => <li key={j}>{e}</li>)}
              </ul>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
