import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';

export default function CommunityPage() {
  const { id } = useParams();
  const { user: me } = useAuth();
  const [c, setC] = useState(null);

  async function load() { setC(await api.getCommunity(id)); }
  useEffect(() => { load(); }, [id]);

  if (!c) return <div className="container"><div className="panel"><div className="panel-body">carregando...</div></div></div>;

  const isMember = c.members.some((m) => m.id === me.id);

  return (
    <div className="container">
      <div>
        <div className="panel">
          {c.photo_url ? <img src={c.photo_url} alt="" style={{ width: '100%' }} /> : <div style={{ width: '100%', height: 200, background: '#eee' }} />}
          <div className="panel-body">
            <h3 style={{ margin: 0, color: 'var(--orkut-text)' }}>{c.name}</h3>
            <div className="small">{c.category} — {c.member_count} membros</div>
            <p style={{ whiteSpace: 'pre-wrap', marginTop: 8 }}>{c.description}</p>
            {!isMember
              ? <button onClick={async () => { await api.joinCommunity(id); await load(); }}>participar</button>
              : me.id === c.owner_id
                ? <span className="small">voce e o dono</span>
                : <button className="secondary" onClick={async () => { await api.leaveCommunity(id); await load(); }}>sair</button>
            }
          </div>
        </div>
      </div>

      <div>
        <div className="panel">
          <div className="panel-header">membros ({c.member_count})</div>
          <div className="panel-body">
            <div className="grid-friends">
              {c.members.map((m) => (
                <div className="item" key={m.id}>
                  <Link to={`/profile/${m.id}`}>
                    {m.photo_url ? <img src={m.photo_url} alt="" /> : <div style={{ width: 60, height: 60, background: '#eee', border: '1px solid var(--orkut-border)' }} />}
                    <div>{m.display_name}</div>
                  </Link>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
