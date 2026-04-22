import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';
import ProfileCard from '../components/ProfileCard.jsx';
import RatingEditor from '../components/RatingEditor.jsx';

export default function Profile() {
  const { id } = useParams();
  const { user: me } = useAuth();
  const [user, setUser] = useState(null);
  const [friends, setFriends] = useState([]);
  const [communities, setCommunities] = useState([]);
  const [testimonials, setTestimonials] = useState([]);
  const [summary, setSummary] = useState(null);
  const [mine, setMine] = useState(null);
  const [pendingFriend, setPendingFriend] = useState(false);

  useEffect(() => {
    Promise.all([
      api.getUser(id).then(setUser),
      api.friendList(id).then(setFriends),
      api.userCommunities(id).then(setCommunities),
      api.listTestimonials(id).then((t) => setTestimonials(t.slice(0, 3))),
      api.ratingSummary(id).then((r) => { setSummary(r.summary); setMine(r.mine); }),
      Number(id) !== me.id ? api.trackVisit(id).catch(() => {}) : null
    ]).catch(() => {});
  }, [id, me.id]);

  async function sendRequest() {
    setPendingFriend(true);
    try {
      await api.friendRequest(id);
    } catch (e) { /* ignore */ }
  }

  if (!user) return <div className="container"><div className="panel"><div className="panel-body">carregando...</div></div></div>;

  const isSelf = Number(id) === me.id;
  const isFriend = friends.some((f) => f.id === me.id);

  return (
    <div className="container">
      <div>
        <ProfileCard user={user} summary={summary} />
        {!isSelf && (
          <div className="panel">
            <div className="panel-header">avalie</div>
            <div className="panel-body">
              <RatingEditor userId={user.id} initial={mine} onChange={() => api.ratingSummary(id).then((r) => { setSummary(r.summary); setMine(r.mine); })} />
            </div>
          </div>
        )}
        {!isSelf && !isFriend && (
          <div className="panel">
            <div className="panel-body">
              <button onClick={sendRequest} disabled={pendingFriend}>
                {pendingFriend ? 'pedido enviado' : 'adicionar como amigo'}
              </button>
            </div>
          </div>
        )}
      </div>

      <div>
        <div className="panel">
          <div className="panel-header">{user.display_name}</div>
          <div className="panel-body">
            {user.bio ? <p style={{ whiteSpace: 'pre-wrap' }}>{user.bio}</p> : <p className="small">sem descricao</p>}
            <ul className="stat-list">
              <li><span>status</span><span>{user.status}</span></li>
              {user.age ? <li><span>idade</span><span>{user.age}</span></li> : null}
              {user.city && <li><span>cidade</span><span>{user.city}</span></li>}
              {user.country && <li><span>pais</span><span>{user.country}</span></li>}
            </ul>
          </div>
        </div>

        <div className="panel">
          <div className="panel-header">amigos ({friends.length})</div>
          <div className="panel-body">
            <div className="grid-friends">
              {friends.slice(0, 9).map((f) => (
                <div className="item" key={f.id}>
                  <Link to={`/profile/${f.id}`}>
                    {f.photo_url ? <img src={f.photo_url} alt="" /> : <div style={{ width: 60, height: 60, background: '#eee', border: '1px solid var(--orkut-border)' }} />}
                    <div>{f.display_name}</div>
                  </Link>
                </div>
              ))}
              {friends.length === 0 && <span className="small">nenhum amigo ainda</span>}
            </div>
          </div>
        </div>

        <div className="panel">
          <div className="panel-header">comunidades ({communities.length})</div>
          <div className="panel-body">
            <div className="grid-communities">
              {communities.slice(0, 9).map((c) => (
                <div className="item" key={c.id}>
                  <Link to={`/community/${c.id}`}>
                    {c.photo_url ? <img src={c.photo_url} alt="" /> : <div style={{ width: 60, height: 60, background: '#eee', border: '1px solid var(--orkut-border)' }} />}
                    <div>{c.name}</div>
                  </Link>
                </div>
              ))}
              {communities.length === 0 && <span className="small">nenhuma comunidade</span>}
            </div>
          </div>
        </div>

        <div className="panel">
          <div className="panel-header">depoimentos</div>
          <div className="panel-body">
            {testimonials.length === 0 && <span className="small">nenhum depoimento. <Link to={`/testimonials/${id}`}>escrever</Link></span>}
            {testimonials.map((t) => (
              <div key={t.id} className="testimonial-item">
                {t.author_photo_url ? <img className="avatar" src={t.author_photo_url} alt="" /> : <div className="avatar" />}
                <div>
                  <div className="who">{t.author_display_name}</div>
                  <div className="when">{new Date(t.created_at).toLocaleString('pt-BR')}</div>
                  <div className="body">{t.body}</div>
                </div>
              </div>
            ))}
            <div style={{ marginTop: 6 }}><Link to={`/testimonials/${id}`}>ver todos</Link></div>
          </div>
        </div>
      </div>
    </div>
  );
}
