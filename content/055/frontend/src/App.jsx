import React, { useEffect, useState, useCallback } from "react";
import { api } from "./api.js";
import AddTargetForm from "./components/AddTargetForm.jsx";
import TargetList from "./components/TargetList.jsx";
import Feed from "./components/Feed.jsx";
import DigestPanel from "./components/DigestPanel.jsx";

export default function App() {
  const [networks, setNetworks] = useState([]);
  const [targets, setTargets] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [error, setError] = useState(null);

  const reloadTargets = useCallback(async () => {
    try {
      const list = await api.listTargets();
      setTargets(list);
      if (list.length && !list.find((t) => t.id === selectedId)) {
        setSelectedId(list[0].id);
      }
    } catch (e) {
      setError(e.message);
    }
  }, [selectedId]);

  useEffect(() => {
    api.networks().then((r) => setNetworks(r.supported)).catch((e) => setError(e.message));
    reloadTargets();
  }, [reloadTargets]);

  useEffect(() => {
    const t = setInterval(reloadTargets, 30_000);
    return () => clearInterval(t);
  }, [reloadTargets]);

  const onAdded = async () => {
    await reloadTargets();
  };

  const onScrape = async (id) => {
    try {
      await api.scrapeNow(id);
      await reloadTargets();
    } catch (e) {
      setError(e.message);
    }
  };

  const onDelete = async (id) => {
    if (!confirm("Stop spying on this handle?")) return;
    await api.deleteTarget(id);
    setSelectedId(null);
    await reloadTargets();
  };

  const selected = targets.find((t) => t.id === selectedId);

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">
          <span className="logo">◉</span>
          <span className="title">STALKR</span>
          <span className="tag">covert handle tracker</span>
        </div>
        <div className="meta">
          {targets.length} target{targets.length === 1 ? "" : "s"} under watch
        </div>
      </header>

      {error && (
        <div className="error" onClick={() => setError(null)}>
          {error} <span className="dismiss">(dismiss)</span>
        </div>
      )}

      <main className="layout">
        <aside className="sidebar">
          <AddTargetForm networks={networks} onAdded={onAdded} onError={setError} />
          <TargetList
            targets={targets}
            selectedId={selectedId}
            onSelect={setSelectedId}
            onScrape={onScrape}
            onDelete={onDelete}
          />
        </aside>

        <section className="content">
          {!selected && <div className="empty">Add a target on the left to begin surveillance.</div>}
          {selected && (
            <>
              <DigestPanel target={selected} />
              <Feed target={selected} />
            </>
          )}
        </section>
      </main>
    </div>
  );
}
