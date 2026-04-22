import React, { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { api } from "../api.js";

const EMPTY = {
  slug: "",
  title: "",
  category: "linux",
  difficulty: "medium",
  time_limit_sec: 600,
  objective: "",
  description: "",
  hints: [],
  setup_script: "#!/bin/bash\n",
  verify_script: "#!/bin/bash\necho OK\n",
  privileged: false,
  enabled: true,
};

export default function AdminChallengeEdit() {
  const { id } = useParams();
  const nav = useNavigate();
  const [form, setForm] = useState(EMPTY);
  const [error, setError] = useState(null);
  const [saving, setSaving] = useState(false);
  const [hintsText, setHintsText] = useState("");

  useEffect(() => {
    if (!id) {
      setForm(EMPTY);
      setHintsText("");
      return;
    }
    api.admin
      .getChallenge(id)
      .then((c) => {
        const hints = typeof c.hints === "string" ? JSON.parse(c.hints || "[]") : c.hints || [];
        setForm({
          ...c,
          hints,
          privileged: !!c.privileged,
          enabled: !!c.enabled,
        });
        setHintsText(hints.join("\n"));
      })
      .catch((e) => setError(e.message));
  }, [id]);

  function set(field, value) {
    setForm((f) => ({ ...f, [field]: value }));
  }

  async function save(e) {
    e.preventDefault();
    setError(null);
    setSaving(true);
    try {
      const body = {
        ...form,
        time_limit_sec: Number(form.time_limit_sec),
        hints: hintsText
          .split("\n")
          .map((l) => l.trim())
          .filter(Boolean),
      };
      if (id) await api.admin.updateChallenge(id, body);
      else await api.admin.createChallenge(body);
      nav("/admin");
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <form className="form edit-challenge" onSubmit={save}>
      <h2>{id ? "Edit challenge" : "New challenge"}</h2>
      {error && <div className="error">{error}</div>}

      <div className="row">
        <label>
          Slug
          <input
            required
            value={form.slug}
            onChange={(e) => set("slug", e.target.value)}
            placeholder="my-new-challenge"
          />
        </label>
        <label>
          Title
          <input
            required
            value={form.title}
            onChange={(e) => set("title", e.target.value)}
          />
        </label>
      </div>

      <div className="row">
        <label>
          Category
          <select
            value={form.category}
            onChange={(e) => set("category", e.target.value)}
          >
            {["linux", "networking", "web", "db", "docker", "security", "observability"].map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </label>
        <label>
          Difficulty
          <select
            value={form.difficulty}
            onChange={(e) => set("difficulty", e.target.value)}
          >
            <option value="easy">easy</option>
            <option value="medium">medium</option>
            <option value="hard">hard</option>
          </select>
        </label>
        <label>
          Time limit (sec)
          <input
            type="number"
            min="60"
            value={form.time_limit_sec}
            onChange={(e) => set("time_limit_sec", e.target.value)}
          />
        </label>
      </div>

      <label>
        Objective (shown to the candidate)
        <textarea
          required
          rows={2}
          value={form.objective}
          onChange={(e) => set("objective", e.target.value)}
        />
      </label>

      <label>
        Description
        <textarea
          rows={3}
          value={form.description}
          onChange={(e) => set("description", e.target.value)}
        />
      </label>

      <label>
        Hints (one per line)
        <textarea
          rows={3}
          value={hintsText}
          onChange={(e) => setHintsText(e.target.value)}
        />
      </label>

      <label>
        Setup script (bash, run as root at container boot — break the system)
        <textarea
          required
          rows={10}
          className="code"
          value={form.setup_script}
          onChange={(e) => set("setup_script", e.target.value)}
        />
      </label>

      <label>
        Verify script (bash, exit 0 = passed)
        <textarea
          required
          rows={8}
          className="code"
          value={form.verify_script}
          onChange={(e) => set("verify_script", e.target.value)}
        />
      </label>

      <div className="row">
        <label className="checkbox">
          <input
            type="checkbox"
            checked={form.privileged}
            onChange={(e) => set("privileged", e.target.checked)}
          />
          Privileged container (only if challenge needs it)
        </label>
        <label className="checkbox">
          <input
            type="checkbox"
            checked={form.enabled}
            onChange={(e) => set("enabled", e.target.checked)}
          />
          Enabled
        </label>
      </div>

      <div className="row">
        <button type="submit" className="btn primary" disabled={saving}>
          {saving ? "Saving…" : id ? "Save" : "Create"}
        </button>
        <button type="button" className="btn ghost" onClick={() => nav("/admin")}>
          Cancel
        </button>
      </div>
    </form>
  );
}
