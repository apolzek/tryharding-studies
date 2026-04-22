import React, { useCallback, useEffect, useState } from "react";
import { api } from "../api.js";

function fmt(ts) {
  const d = new Date(ts);
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return `${Math.floor(diff)}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return d.toLocaleString();
}

export default function Feed({ target }) {
  const [events, setEvents] = useState([]);
  const [filter, setFilter] = useState("all");

  const load = useCallback(async () => {
    if (!target) return;
    const list = await api.events(target.id, {
      limit: 150,
      network: filter === "all" ? undefined : filter,
    });
    setEvents(list);
  }, [target, filter]);

  useEffect(() => {
    load();
    const t = setInterval(load, 30_000);
    return () => clearInterval(t);
  }, [load]);

  const filters = ["all", ...(target?.networks || [])];

  return (
    <section className="feed">
      <div className="feed-head">
        <h2>Activity feed</h2>
        <div className="feed-filters">
          {filters.map((n) => (
            <button
              key={n}
              className={filter === n ? "on" : ""}
              onClick={() => setFilter(n)}
            >
              {n}
            </button>
          ))}
        </div>
      </div>
      {!events.length && <div className="empty">No events captured yet.</div>}
      <ul className="events">
        {events.map((e) => (
          <li key={e.id} className={`event net-${e.network}`}>
            <div className="event-head">
              <span className={`pill ${e.network}`}>{e.network}</span>
              <span className="kind">{e.kind}</span>
              <span className="when">{fmt(e.happened_at)}</span>
            </div>
            <a className="title" href={e.url} target="_blank" rel="noreferrer">
              {e.title}
            </a>
            {e.body && <div className="body">{e.body.slice(0, 280)}{e.body.length > 280 ? "…" : ""}</div>}
          </li>
        ))}
      </ul>
    </section>
  );
}
