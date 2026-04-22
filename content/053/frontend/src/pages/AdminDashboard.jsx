import React, { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api.js";

export default function AdminDashboard() {
  const [tab, setTab] = useState("challenges");
  const [challenges, setChallenges] = useState([]);
  const [sessions, setSessions] = useState([]);
  const [error, setError] = useState(null);
  const nav = useNavigate();

  async function load() {
    try {
      const [c, s] = await Promise.all([
        api.admin.listChallenges(),
        api.admin.listSessions(),
      ]);
      setChallenges(c);
      setSessions(s);
    } catch (e) {
      setError(e.message);
      if (/token|401|admin/i.test(e.message)) {
        localStorage.removeItem("sre_token");
        nav("/admin/login");
      }
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function remove(id) {
    if (!confirm("Delete this challenge?")) return;
    await api.admin.deleteChallenge(id);
    load();
  }

  function logout() {
    localStorage.removeItem("sre_token");
    localStorage.removeItem("sre_user");
    nav("/admin/login");
  }

  return (
    <div className="admin-page">
      <div className="admin-header">
        <h2>Admin</h2>
        <div>
          <Link to="/admin/challenges/new" className="btn primary">
            + New challenge
          </Link>
          <button className="btn ghost" onClick={logout}>
            Logout
          </button>
        </div>
      </div>

      <div className="tabs">
        <button
          className={tab === "challenges" ? "active" : ""}
          onClick={() => setTab("challenges")}
        >
          Challenges ({challenges.length})
        </button>
        <button
          className={tab === "sessions" ? "active" : ""}
          onClick={() => setTab("sessions")}
        >
          Sessions ({sessions.length})
        </button>
      </div>

      {error && <div className="error">{error}</div>}

      {tab === "challenges" && (
        <table className="table">
          <thead>
            <tr>
              <th>#</th>
              <th>Slug</th>
              <th>Title</th>
              <th>Category</th>
              <th>Difficulty</th>
              <th>Enabled</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {challenges.map((c) => (
              <tr key={c.id}>
                <td>{c.id}</td>
                <td>
                  <code>{c.slug}</code>
                </td>
                <td>{c.title}</td>
                <td>{c.category}</td>
                <td>
                  <span className={`tag tag-${c.difficulty}`}>
                    {c.difficulty}
                  </span>
                </td>
                <td>{c.enabled ? "✓" : "✗"}</td>
                <td>
                  <Link to={`/admin/challenges/${c.id}`}>edit</Link>{" "}
                  <button className="link danger" onClick={() => remove(c.id)}>
                    delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {tab === "sessions" && (
        <table className="table">
          <thead>
            <tr>
              <th>Session</th>
              <th>Challenge</th>
              <th>Status</th>
              <th>Passed</th>
              <th>Port</th>
              <th>Started</th>
            </tr>
          </thead>
          <tbody>
            {sessions.map((s) => (
              <tr key={s.id}>
                <td>
                  <code>{s.id.slice(0, 8)}</code>
                </td>
                <td>{s.title}</td>
                <td>{s.status}</td>
                <td>{s.passed ? "✅" : "—"}</td>
                <td>{s.port}</td>
                <td>{s.started_at}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
