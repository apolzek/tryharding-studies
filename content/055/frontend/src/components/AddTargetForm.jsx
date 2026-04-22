import React, { useState } from "react";
import { api } from "../api.js";

export default function AddTargetForm({ networks, onAdded, onError }) {
  const [handle, setHandle] = useState("");
  const [picked, setPicked] = useState(new Set(["github"]));
  const [busy, setBusy] = useState(false);

  const toggle = (n) => {
    const next = new Set(picked);
    if (next.has(n)) next.delete(n);
    else next.add(n);
    setPicked(next);
  };

  const submit = async (e) => {
    e.preventDefault();
    if (!handle.trim()) return;
    setBusy(true);
    try {
      await api.createTarget(handle.trim(), [...picked]);
      setHandle("");
      await onAdded();
    } catch (err) {
      onError(err.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className="add-form" onSubmit={submit}>
      <label className="field">
        <span>Target handle</span>
        <div className="handle-input">
          <span className="at">@</span>
          <input
            value={handle}
            placeholder="apolzek"
            onChange={(e) => setHandle(e.target.value.replace(/^@/, ""))}
            disabled={busy}
          />
        </div>
      </label>

      <div className="field">
        <span>Networks to monitor</span>
        <div className="chips">
          {networks.map((n) => (
            <label key={n} className={`chip ${picked.has(n) ? "on" : ""}`}>
              <input
                type="checkbox"
                checked={picked.has(n)}
                onChange={() => toggle(n)}
                disabled={busy}
              />
              {n}
            </label>
          ))}
        </div>
      </div>

      <button type="submit" disabled={busy || !handle.trim() || picked.size === 0}>
        {busy ? "deploying…" : "Deploy surveillance"}
      </button>
    </form>
  );
}
