import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';

export default function Testimonials() {
  const { id } = useParams();
  const { user: me } = useAuth();
  const [owner, setOwner] = useState(null);
  const [list, setList] = useState([]);
  const [body, setBody] = useState('');
  const [error, setError] = useState('');

  async function load() {
    const [u, t] = await Promise.all([api.getUser(id), api.listTestimonials(id)]);
    setOwner(u);
    setList(t);
  }

  useEffect(() => { load(); }, [id]);

  async function submit(e) {
    e.preventDefault();
    setError('');
    try {
      await api.postTestimonial(id, body);
      setBody('');
      await load();
    } catch (err) { setError(err.message || 'erro'); }
  }

  async function remove(tid) {
    await api.deleteTestimonial(tid);
    await load();
  }

  return (
    <div className="container single">
      <div className="panel">
        <div className="panel-header">depoimentos de {owner?.display_name}</div>
        <div className="panel-body">
          {Number(id) !== me.id && (
            <form onSubmit={submit} style={{ marginBottom: 12 }}>
              <textarea rows="3" value={body} onChange={(e) => setBody(e.target.value)} placeholder="escreva um depoimento..." />
              {error && <div className="error">{error}</div>}
              <div style={{ marginTop: 4 }}>
                <button type="submit">publicar</button>
              </div>
            </form>
          )}
          {list.length === 0 && <span className="small">nenhum depoimento ainda</span>}
          {list.map((t) => (
            <div key={t.id} className="testimonial-item">
              {t.author_photo_url ? <img className="avatar" src={t.author_photo_url} alt="" /> : <div className="avatar" />}
              <div style={{ flex: 1 }}>
                <div className="who"><Link to={`/profile/${t.author_user_id}`}>{t.author_display_name}</Link></div>
                <div className="when">{new Date(t.created_at).toLocaleString('pt-BR')}</div>
                <div className="body">{t.body}</div>
                {(t.author_user_id === me.id || Number(id) === me.id) && (
                  <button className="secondary" style={{ marginTop: 4 }} onClick={() => remove(t.id)}>apagar</button>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
