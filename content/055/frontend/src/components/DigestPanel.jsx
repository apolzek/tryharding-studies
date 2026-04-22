import React, { useCallback, useEffect, useState } from "react";
import { api } from "../api.js";

export default function DigestPanel({ target }) {
  const [hours, setHours] = useState(24);
  const [digest, setDigest] = useState(null);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    if (!target) return;
    setLoading(true);
    try {
      setDigest(await api.digest(target.id, hours));
    } finally {
      setLoading(false);
    }
  }, [target, hours]);

  useEffect(() => {
    load();
    const t = setInterval(load, 60_000);
    return () => clearInterval(t);
  }, [load]);

  if (!digest) return <div className="digest">{loading ? "compiling digest…" : "no digest yet"}</div>;

  const countsByNet = digest.by_network_counts || {};
  const nets = Object.keys(countsByNet);

  return (
    <section className="digest">
      <div className="digest-head">
        <h2>Daily digest — @{digest.target}</h2>
        <div className="window-picker">
          window:
          {[6, 24, 72, 168].map((h) => (
            <button
              key={h}
              className={h === hours ? "on" : ""}
              onClick={() => setHours(h)}
            >
              {h}h
            </button>
          ))}
        </div>
      </div>
      <div className="digest-stats">
        <div className="stat big">
          <span className="num">{digest.total_events}</span>
          <span className="lbl">events in {digest.window_hours}h</span>
        </div>
        {nets.map((n) => (
          <div key={n} className="stat">
            <span className="num">{countsByNet[n]}</span>
            <span className="lbl">{n}</span>
          </div>
        ))}
      </div>
      {digest.github_summary && (
        <pre className="github-summary">{digest.github_summary}</pre>
      )}
    </section>
  );
}
