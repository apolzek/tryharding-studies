import React from 'react';
import { NavLink, Link, useLocation } from 'react-router-dom';
import { useAuth } from '../auth.jsx';

export default function Header() {
  const { user, logout } = useAuth();
  const loc = useLocation();
  if (!user) return null;
  return (
    <div>
      <div className="top-bar">
        <div className="logo">orkut<span>.</span></div>
        <div style={{ flex: 1 }} />
        <span className="small">ola, <strong>{user.display_name}</strong></span>
        <Link to="/profile/edit" className="small">editar perfil</Link>
        <button className="secondary" onClick={logout}>sair</button>
      </div>
      <div className="nav-tabs">
        <NavLink to="/home" className={({ isActive }) => (isActive ? 'active' : '')}>home</NavLink>
        <NavLink to={`/profile/${user.id}`} className={loc.pathname.startsWith(`/profile/${user.id}`) ? 'active' : ''}>perfil</NavLink>
        <NavLink to={`/scrapbook/${user.id}`} className={loc.pathname.startsWith('/scrapbook') ? 'active' : ''}>scrapbook</NavLink>
        <NavLink to="/friends" className={loc.pathname.startsWith('/friends') ? 'active' : ''}>amigos</NavLink>
        <NavLink to="/communities" className={loc.pathname.startsWith('/community') || loc.pathname.startsWith('/communities') ? 'active' : ''}>comunidades</NavLink>
        <NavLink to={`/testimonials/${user.id}`} className={loc.pathname.startsWith('/testimonials') ? 'active' : ''}>depoimentos</NavLink>
        <NavLink to={`/photos/${user.id}`} className={loc.pathname.startsWith('/photos') ? 'active' : ''}>fotos</NavLink>
      </div>
    </div>
  );
}
