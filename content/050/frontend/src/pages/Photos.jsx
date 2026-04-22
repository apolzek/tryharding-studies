import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';

export default function Photos() {
  const { id } = useParams();
  const { user: me } = useAuth();
  const [list, setList] = useState([]);
  const [url, setUrl] = useState('');
  const [caption, setCaption] = useState('');

  async function load() { setList(await api.listPhotos(id)); }
  useEffect(() => { load(); }, [id]);

  async function add(e) {
    e.preventDefault();
    if (!url.trim()) return;
    await api.addPhoto(url, caption);
    setUrl(''); setCaption('');
    await load();
  }

  return (
    <div className="container single">
      <div className="panel">
        <div className="panel-header">album</div>
        <div className="panel-body">
          {Number(id) === me.id && (
            <form onSubmit={add} style={{ marginBottom: 10 }}>
              <div className="form-row"><label>url da foto</label><input value={url} onChange={(e) => setUrl(e.target.value)} /></div>
              <div className="form-row"><label>legenda</label><input value={caption} onChange={(e) => setCaption(e.target.value)} /></div>
              <button type="submit">adicionar</button>
            </form>
          )}
          <div className="photo-grid">
            {list.map((p) => (
              <div key={p.id}>
                <img src={p.url} alt={p.caption} title={p.caption} />
                {Number(id) === me.id && <button className="secondary" style={{ fontSize: 10 }} onClick={async () => { await api.deletePhoto(p.id); await load(); }}>apagar</button>}
              </div>
            ))}
          </div>
          {list.length === 0 && <span className="small">nenhuma foto</span>}
        </div>
      </div>
    </div>
  );
}
