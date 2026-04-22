import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth.jsx';

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  async function onSubmit(e) {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await login(username, password);
      navigate('/home');
    } catch (err) {
      setError(err.message || 'erro ao entrar');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="login-wrapper">
      <div className="panel-header">
        <span className="logo" style={{ fontSize: 18, color: '#fff' }}>orkut<span style={{ color: '#fff' }}>.</span></span>
        <span style={{ float: 'right' }}>conecte-se aos seus amigos</span>
      </div>
      <div className="panel-body">
        <form onSubmit={onSubmit} aria-label="login-form">
          <div className="form-row">
            <label htmlFor="username">usuario</label>
            <input id="username" type="text" value={username} onChange={(e) => setUsername(e.target.value)} />
          </div>
          <div className="form-row">
            <label htmlFor="password">senha</label>
            <input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          {error && <div className="error" role="alert">{error}</div>}
          <div style={{ marginTop: 10 }}>
            <button type="submit" disabled={busy}>entrar</button>
            <span style={{ marginLeft: 10 }}>
              ainda nao tem conta? <Link to="/register">cadastre-se</Link>
            </span>
          </div>
        </form>
      </div>
    </div>
  );
}
