import React from "react";

function fmt(ts) {
  if (!ts) return "never";
  const d = new Date(ts);
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return `${Math.floor(diff)}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return d.toLocaleString();
}

export default function TargetList({ targets, selectedId, onSelect, onScrape, onDelete }) {
  if (!targets.length) return <div className="target-list empty-list">No targets yet.</div>;
  return (
    <ul className="target-list">
      {targets.map((t) => (
        <li
          key={t.id}
          className={`target ${t.id === selectedId ? "selected" : ""}`}
          onClick={() => onSelect(t.id)}
        >
          <div className="row">
            <span className="handle">@{t.handle}</span>
            <span className="badge-count">{t.networks.length}</span>
          </div>
          <div className="networks-mini">
            {t.networks.map((n) => (
              <span key={n} className="mini-chip">
                {n}
              </span>
            ))}
          </div>
          <div className="target-meta">
            <span title="last scraped">⟳ {fmt(t.last_scraped_at)}</span>
            <span className="actions">
              <button
                className="ghost"
                onClick={(e) => {
                  e.stopPropagation();
                  onScrape(t.id);
                }}
              >
                scrape
              </button>
              <button
                className="ghost danger"
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(t.id);
                }}
              >
                ✕
              </button>
            </span>
          </div>
        </li>
      ))}
    </ul>
  );
}
