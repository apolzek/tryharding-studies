import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api.js';

export default function Communities() {
  const [mine, setMine] = useState([]);
  const [all, setAll] = useState([]);
  const [q, setQ] = useState('');
  const [form, setForm] = useState({ name: '', description: '', category: '', photo_url: '' });
  const [creating, setCreating] = useState(false);

  async function load() {
    setMine(await api.myCommunities());
    setAll(await api.listCommunities(q));
  }
  useEffect(() => { load(); }, []);

  async function search(e) {
    e.preventDefault();
    setAll(await api.listCommunities(q));
  }

  async function create(e) {
    e.preventDefault();
    if (!form.name.trim()) return;
    setCreating(true);
    try {
      await api.createCommunity(form);
      setForm({ name: '', description: '', category: '', photo_url: '' });
      await load();
    } finally { setCreating(false); }
  }

  return (
    <div className="container">
      <div>
        <div className="panel">
          <div className="panel-header">minhas comunidades ({mine.length})</div>
          <div className="panel-body">
            <div className="grid-communities">
              {mine.map((c) => (
                <div className="item" key={c.id}>
                  <Link to={`/community/${c.id}`}>
                    {c.photo_url ? <img src={c.photo_url} alt="" /> : <div style={{ width: 60, height: 60, background: '#eee', border: '1px solid var(--orkut-border)' }} />}
                    <div>{c.name}</div>
                  </Link>
                </div>
              ))}
              {mine.length === 0 && <span className="small">nenhuma comunidade</span>}
            </div>
          </div>
        </div>

        <div className="panel">
          <div className="panel-header">criar comunidade</div>
          <div className="panel-body">
            <form onSubmit={create}>
              <div className="form-row"><label>nome</label><input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></div>
              <div className="form-row"><label>descricao</label><textarea rows="3" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></div>
              <div className="form-row"><label>categoria</label><input value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} /></div>
              <div className="form-row"><label>foto (url)</label><input value={form.photo_url} onChange={(e) => setForm({ ...form, photo_url: e.target.value })} /></div>
              <button type="submit" disabled={creating}>criar</button>
            </form>
          </div>
        </div>
      </div>

      <div>
        <div className="panel">
          <div className="panel-header">procurar comunidades</div>
          <div className="panel-body">
            <form onSubmit={search} className="search-box">
              <input type="text" value={q} onChange={(e) => setQ(e.target.value)} placeholder="nome, descricao, categoria..." />
              <button type="submit">buscar</button>
            </form>
          </div>
        </div>

        <div className="panel">
          <div className="panel-header">todas comunidades</div>
          <div className="panel-body">
            {all.map((c) => (
              <div key={c.id} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '6px 0', borderBottom: '1px dashed #ccc' }}>
                {c.photo_url ? <img src={c.photo_url} alt="" style={{ width: 48, height: 48, border: '1px solid var(--orkut-border)' }} /> : <div style={{ width: 48, height: 48, background: '#eee' }} />}
                <div style={{ flex: 1 }}>
                  <div><Link to={`/community/${c.id}`}><strong>{c.name}</strong></Link></div>
                  <div className="small">{c.category} — {c.member_count} membros</div>
                  <div className="small">{c.description}</div>
                </div>
              </div>
            ))}
            {all.length === 0 && <span className="small">nenhuma comunidade</span>}
          </div>
        </div>
      </div>
    </div>
  );
}
