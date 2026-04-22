import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';

export default function EditProfile() {
  const { user, refresh } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({
    display_name: '', photo_url: '', bio: '', status: 'solteiro(a)', age: 0, city: '', country: 'Brasil'
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!user) return;
    setForm({
      display_name: user.display_name || '',
      photo_url: user.photo_url || '',
      bio: user.bio || '',
      status: user.status || 'solteiro(a)',
      age: user.age || 0,
      city: user.city || '',
      country: user.country || 'Brasil'
    });
  }, [user]);

  function set(k, v) { setForm((f) => ({ ...f, [k]: v })); }

  async function onSubmit(e) {
    e.preventDefault();
    setSaving(true);
    setError('');
    try {
      await api.updateMe({ ...form, age: Number(form.age) || 0 });
      await refresh();
      navigate(`/profile/${user.id}`);
    } catch (err) {
      setError(err.message || 'erro ao salvar');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="container single">
      <div className="panel">
        <div className="panel-header">editar perfil</div>
        <div className="panel-body">
          <form onSubmit={onSubmit}>
            <div className="form-row">
              <label>nome exibido</label>
              <input type="text" value={form.display_name} onChange={(e) => set('display_name', e.target.value)} />
            </div>
            <div className="form-row">
              <label>url da foto</label>
              <input type="text" value={form.photo_url} onChange={(e) => set('photo_url', e.target.value)} placeholder="https://..." />
            </div>
            <div className="form-row">
              <label>sobre mim</label>
              <textarea rows="4" value={form.bio} onChange={(e) => set('bio', e.target.value)} />
            </div>
            <div className="form-row">
              <label>status</label>
              <select value={form.status} onChange={(e) => set('status', e.target.value)}>
                <option>solteiro(a)</option>
                <option>namorando</option>
                <option>casado(a)</option>
                <option>em um relacionamento aberto</option>
                <option>e complicado</option>
              </select>
            </div>
            <div className="form-row">
              <label>idade</label>
              <input type="number" value={form.age} onChange={(e) => set('age', e.target.value)} />
            </div>
            <div className="form-row">
              <label>cidade</label>
              <input type="text" value={form.city} onChange={(e) => set('city', e.target.value)} />
            </div>
            <div className="form-row">
              <label>pais</label>
              <input type="text" value={form.country} onChange={(e) => set('country', e.target.value)} />
            </div>
            {error && <div className="error">{error}</div>}
            <button type="submit" disabled={saving}>salvar</button>
          </form>
        </div>
      </div>
    </div>
  );
}
