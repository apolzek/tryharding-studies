import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';

export default function Friends() {
  const { user: me } = useAuth();
  const [friends, setFriends] = useState([]);
  const [pending, setPending] = useState([]);
  const [q, setQ] = useState('');
  const [results, setResults] = useState([]);

  async function reload() {
    setFriends(await api.friendList(me.id));
    setPending(await api.friendPending());
  }
  useEffect(() => { reload(); }, [me.id]);

  async function search(e) {
    e.preventDefault();
    if (!q.trim()) { setResults([]); return; }
    setResults(await api.searchUsers(q));
  }

  async function addFriend(uid) {
    try { await api.friendRequest(uid); } catch (e) {}
    await reload();
  }

  return (
    <div className="container single">
      {pending.length > 0 && (
        <div className="panel">
          <div className="panel-header">pedidos pendentes</div>
          <div className="panel-body">
            {pending.map((p) => (
              <div key={p.id} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0' }}>
                {p.photo_url ? <img src={p.photo_url} alt="" style={{ width: 40, height: 40, border: '1px solid var(--orkut-border)' }} /> : <div style={{ width: 40, height: 40, background: '#eee' }} />}
                <Link to={`/profile/${p.id}`}>{p.display_name}</Link>
                <div style={{ flex: 1 }} />
                <button onClick={async () => { await api.friendAccept(p.id); await reload(); }}>aceitar</button>
                <button className="secondary" onClick={async () => { await api.friendRemove(p.id); await reload(); }}>recusar</button>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="panel">
        <div className="panel-header">procurar pessoas</div>
        <div className="panel-body">
          <form onSubmit={search} className="search-box">
            <input type="text" value={q} onChange={(e) => setQ(e.target.value)} placeholder="nome ou usuario..." />
            <button type="submit">buscar</button>
          </form>
          <div style={{ marginTop: 8 }}>
            {results.filter((u) => u.id !== me.id).map((u) => (
              <div key={u.id} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', borderBottom: '1px dashed #ccc' }}>
                {u.photo_url ? <img src={u.photo_url} alt="" style={{ width: 40, height: 40, border: '1px solid var(--orkut-border)' }} /> : <div style={{ width: 40, height: 40, background: '#eee' }} />}
                <Link to={`/profile/${u.id}`}>{u.display_name}</Link>
                <div style={{ flex: 1 }} />
                {!friends.some((f) => f.id === u.id) && (
                  <button onClick={() => addFriend(u.id)}>adicionar</button>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="panel">
        <div className="panel-header">meus amigos ({friends.length})</div>
        <div className="panel-body">
          <div className="grid-friends">
            {friends.map((f) => (
              <div className="item" key={f.id}>
                <Link to={`/profile/${f.id}`}>
                  {f.photo_url ? <img src={f.photo_url} alt="" /> : <div style={{ width: 60, height: 60, background: '#eee', border: '1px solid var(--orkut-border)' }} />}
                  <div>{f.display_name}</div>
                </Link>
                <button className="secondary" style={{ fontSize: 10, marginTop: 2 }} onClick={async () => { await api.friendRemove(f.id); await reload(); }}>remover</button>
              </div>
            ))}
            {friends.length === 0 && <span className="small">voce nao tem amigos ainda</span>}
          </div>
        </div>
      </div>
    </div>
  );
}
