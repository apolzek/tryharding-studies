import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';

export default function Scrapbook() {
  const { id } = useParams();
  const { user: me } = useAuth();
  const [owner, setOwner] = useState(null);
  const [scraps, setScraps] = useState([]);
  const [body, setBody] = useState('');
  const [busy, setBusy] = useState(false);

  async function load() {
    const [u, s] = await Promise.all([api.getUser(id), api.listScraps(id)]);
    setOwner(u);
    setScraps(s);
  }

  useEffect(() => { load(); }, [id]);

  async function submit(e) {
    e.preventDefault();
    if (!body.trim()) return;
    setBusy(true);
    try {
      await api.postScrap(id, body);
      setBody('');
      await load();
    } finally { setBusy(false); }
  }

  async function remove(scrapId) {
    await api.deleteScrap(scrapId);
    await load();
  }

  return (
    <div className="container single">
      <div className="panel">
        <div className="panel-header">scrapbook de {owner?.display_name}</div>
        <div className="panel-body">
          {Number(id) !== me.id && (
            <form onSubmit={submit} style={{ marginBottom: 12 }}>
              <textarea rows="3" value={body} onChange={(e) => setBody(e.target.value)} placeholder="escreva um scrap..." />
              <div style={{ marginTop: 4 }}>
                <button type="submit" disabled={busy}>enviar</button>
              </div>
            </form>
          )}
          {scraps.length === 0 && <span className="small">nenhum scrap ainda</span>}
          {scraps.map((s) => (
            <div key={s.id} className="scrap-item">
              {s.author_photo_url ? <img className="avatar" src={s.author_photo_url} alt="" /> : <div className="avatar" />}
              <div style={{ flex: 1 }}>
                <div className="who"><Link to={`/profile/${s.author_user_id}`}>{s.author_display_name}</Link></div>
                <div className="when">{new Date(s.created_at).toLocaleString('pt-BR')}</div>
                <div className="body">{s.body}</div>
                {(s.author_user_id === me.id || Number(id) === me.id) && (
                  <button className="secondary" style={{ marginTop: 4 }} onClick={() => remove(s.id)}>apagar</button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
