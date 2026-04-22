import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api.js";

export default function AdminLogin() {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(null);
  const nav = useNavigate();

  async function submit(e) {
    e.preventDefault();
    setError(null);
    try {
      const r = await api.login(username, password);
      localStorage.setItem("sre_token", r.token);
      localStorage.setItem("sre_user", JSON.stringify(r.user));
      nav("/admin");
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div className="login-page">
      <form className="card form" onSubmit={submit}>
        <h2>Admin login</h2>
        <label>
          Username
          <input value={username} onChange={(e) => setUsername(e.target.value)} />
        </label>
        <label>
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        {error && <div className="error">{error}</div>}
        <button type="submit" className="btn primary">
          Log in
        </button>
        <p className="small">Default: admin / admin123 (change via env var).</p>
      </form>
    </div>
  );
}
