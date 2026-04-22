import React from 'react';
import { Link } from 'react-router-dom';

export default function ProfileCard({ user, summary }) {
  if (!user) return null;
  return (
    <div className="panel profile-card">
      {user.photo_url
        ? <img src={user.photo_url} alt={user.display_name} className="photo" />
        : <div className="photo-empty">sem foto</div>}
      <div className="name">
        <Link to={`/profile/${user.id}`}>{user.display_name}</Link>
      </div>
      <div className="meta">
        {user.status && <div>{user.status}</div>}
        {user.age ? <div>{user.age} anos</div> : null}
        {(user.city || user.country) && <div>{[user.city, user.country].filter(Boolean).join(', ')}</div>}
      </div>
      {summary && (
        <div className="panel-body" style={{ borderTop: '1px solid var(--orkut-border)' }}>
          <RatingLine label="fas" text={`${summary.fans} fas`} />
          <RatingLine label="confiavel" n={summary.trust} icon="☺" />
          <RatingLine label="legal" n={summary.cool} icon="❄" />
          <RatingLine label="sexy" n={summary.sexy} icon="♥" />
        </div>
      )}
    </div>
  );
}

function RatingLine({ label, n, icon, text }) {
  if (text) {
    return (
      <div className="rating-row">
        <span className="label">{label}</span>
        <span className="icons">{text}</span>
      </div>
    );
  }
  const filled = Math.round(n || 0);
  return (
    <div className="rating-row">
      <span className="label">{label}</span>
      <span className="icons">
        {[0, 1, 2].map((i) => (
          <span key={i} className={i < filled ? 'icon-full' : 'icon-empty'}>{icon}</span>
        ))}
      </span>
    </div>
  );
}
