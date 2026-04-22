import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth.jsx';

export default function Register() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [display, setDisplay] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  async function onSubmit(e) {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await register(username, password, display || username);
      navigate('/home');
    } catch (err) {
      setError(err.message || 'erro ao cadastrar');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="login-wrapper">
      <div className="panel-header">criar conta</div>
      <div className="panel-body">
        <form onSubmit={onSubmit} aria-label="register-form">
          <div className="form-row">
            <label htmlFor="display">nome completo</label>
            <input id="display" type="text" value={display} onChange={(e) => setDisplay(e.target.value)} />
          </div>
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
            <button type="submit" disabled={busy}>cadastrar</button>
            <span style={{ marginLeft: 10 }}>
              ja tem conta? <Link to="/login">entre</Link>
            </span>
          </div>
        </form>
      </div>
    </div>
  );
}
