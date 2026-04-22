import React, { useState } from 'react';
import { api } from '../api.js';

const DIMENSIONS = [
  { key: 'trust', label: 'confiavel', icon: '☺' },
  { key: 'cool', label: 'legal', icon: '❄' },
  { key: 'sexy', label: 'sexy', icon: '♥' }
];

export default function RatingEditor({ userId, initial, onChange }) {
  const [state, setState] = useState({
    trust: initial?.trust ?? 0,
    cool: initial?.cool ?? 0,
    sexy: initial?.sexy ?? 0,
    is_fan: !!initial?.is_fan
  });
  const [saving, setSaving] = useState(false);

  async function save(next) {
    setSaving(true);
    try {
      const saved = await api.rate(userId, next);
      setState(saved);
      if (onChange) onChange(saved);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      {DIMENSIONS.map((d) => (
        <div key={d.key} className="rating-row">
          <span className="label">{d.label}</span>
          <span className="icons">
            {[1, 2, 3].map((n) => (
              <button
                key={n}
                className="secondary"
                style={{ padding: '0 4px', marginRight: 2 }}
                disabled={saving}
                onClick={() => save({ ...state, [d.key]: n })}
                aria-label={`${d.label} ${n}`}
              >
                <span className={state[d.key] >= n ? 'icon-full' : 'icon-empty'}>{d.icon}</span>
              </button>
            ))}
            <button className="secondary" style={{ padding: '0 4px' }} disabled={saving}
              onClick={() => save({ ...state, [d.key]: 0 })}>x</button>
          </span>
        </div>
      ))}
      <div style={{ marginTop: 6 }}>
        <button
          className={state.is_fan ? '' : 'secondary'}
          disabled={saving}
          onClick={() => save({ ...state, is_fan: !state.is_fan })}
        >
          {state.is_fan ? '♥ sou fa' : '♡ virar fa'}
        </button>
      </div>
    </div>
  );
}
