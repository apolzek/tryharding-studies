import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../api.js';

export default function Rights() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [list, setList] = useState(null);
  const [detail, setDetail] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    setError(null);
    if (id) {
      setDetail(null);
      api.rightsDetail(id).then(setDetail).catch((e) => setError(e.message));
    } else {
      setDetail(null);
      if (!list) api.rightsList().then((d) => setList(d.categories)).catch((e) => setError(e.message));
    }
  }, [id]);

  if (error) return <div className="error">Erro: {error}</div>;

  if (id) {
    if (!detail) return <div className="loading">Carregando…</div>;
    return (
      <div>
        <div className="breadcrumb">
          <Link to="/direitos">← Direitos & Normas</Link> / {detail.title}
        </div>
        <div className="card" style={{ borderTopColor: 'var(--azul)' }}>
          <h2 style={{ color: 'var(--azul)' }}>{detail.title}</h2>
          <p style={{ color: 'var(--muted)' }}>{detail.description}</p>
          {detail.items.map((item, i) => (
            <div key={i} className="item">
              <h4>{item.title}</h4>
              <p style={{ margin: 0 }}>{item.text}</p>
              {item.source && <span className="source">Fonte: {item.source}</span>}
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!list) return <div className="loading">Carregando categorias…</div>;

  return (
    <div>
      <h2>Direitos & Normas</h2>
      <p style={{ color: 'var(--muted)' }}>
        O que todo brasileiro precisa saber. Do Código de Defesa do Consumidor à CLT, do Detran ao SUS.
      </p>
      <div className="grid">
        {list.map((c) => (
          <div key={c.id} className="card category-card" onClick={() => navigate(`/direitos/${c.id}`)}>
            <h3>{c.title}</h3>
            <p style={{ color: 'var(--muted)' }}>{c.description}</p>
            <span className="pill">{c.count} tópicos</span>
          </div>
        ))}
      </div>
    </div>
  );
}
