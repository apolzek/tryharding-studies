import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../auth.jsx';
import { api } from '../api.js';
import ProfileCard from '../components/ProfileCard.jsx';

export default function Home() {
  const { user } = useAuth();
  const [friends, setFriends] = useState([]);
  const [communities, setCommunities] = useState([]);
  const [pending, setPending] = useState([]);
  const [scraps, setScraps] = useState([]);
  const [visitors, setVisitors] = useState([]);
  const [summary, setSummary] = useState(null);

  useEffect(() => {
    if (!user) return;
    Promise.all([
      api.friendList(user.id).then(setFriends),
      api.myCommunities().then(setCommunities),
      api.friendPending().then(setPending),
      api.listScraps(user.id).then((s) => setScraps(s.slice(0, 5))),
      api.listVisits(user.id).then(setVisitors),
      api.ratingSummary(user.id).then((r) => setSummary(r.summary))
    ]).catch(() => {});
  }, [user]);

  return (
    <div className="container">
      <div>
        <ProfileCard user={user} summary={summary} />
        <div className="panel">
          <div className="panel-header">visitantes recentes</div>
          <div className="panel-body visitors-list">
            {visitors.length === 0 && <span className="small">nenhuma visita ainda</span>}
            {visitors.map((v) => (
              <Link key={v.id} to={`/profile/${v.id}`} title={v.display_name}>
                {v.photo_url ? <img src={v.photo_url} alt={v.display_name} /> : <div style={{ width: 32, height: 32, background: '#ddd' }} />}
              </Link>
            ))}
          </div>
        </div>
      </div>

      <div>
        <div className="panel">
          <div className="panel-header">bem-vindo(a), {user.display_name}</div>
          <div className="panel-body">
            <ul className="stat-list">
              <li><Link to="/friends">meus amigos</Link><span className="count">{friends.length}</span></li>
              <li><Link to="/communities">minhas comunidades</Link><span className="count">{communities.length}</span></li>
              <li><Link to={`/scrapbook/${user.id}`}>scraps</Link><span className="count">{scraps.length}</span></li>
              <li><Link to="/friends">pedidos de amizade</Link><span className="count">{pending.length}</span></li>
            </ul>
          </div>
        </div>

        {pending.length > 0 && (
          <div className="panel">
            <div className="panel-header">novos pedidos de amizade</div>
            <div className="panel-body">
              {pending.map((p) => (
                <div key={p.id} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0' }}>
                  {p.photo_url ? <img src={p.photo_url} alt="" style={{ width: 32, height: 32, border: '1px solid var(--orkut-border)' }} /> : <div style={{ width: 32, height: 32, background: '#ddd' }} />}
                  <Link to={`/profile/${p.id}`}>{p.display_name}</Link>
                  <div style={{ flex: 1 }} />
                  <button onClick={async () => { await api.friendAccept(p.id); setPending(await api.friendPending()); }}>aceitar</button>
                  <button className="secondary" onClick={async () => { await api.friendRemove(p.id); setPending(await api.friendPending()); }}>recusar</button>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="panel">
          <div className="panel-header">scraps recentes</div>
          <div className="panel-body">
            {scraps.length === 0 && <span className="small">nenhum scrap ainda. <Link to="/friends">encontre amigos</Link></span>}
            {scraps.map((s) => (
              <div key={s.id} className="scrap-item">
                {s.author_photo_url ? <img className="avatar" src={s.author_photo_url} alt="" /> : <div className="avatar" />}
                <div>
                  <div className="who"><Link to={`/profile/${s.author_user_id}`}>{s.author_display_name}</Link></div>
                  <div className="when">{new Date(s.created_at).toLocaleString('pt-BR')}</div>
                  <div className="body">{s.body}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
