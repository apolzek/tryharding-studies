import React from "react";
import { Link } from "react-router-dom";

export default function Home() {
  return (
    <div className="hero">
      <h1>Hands-on SRE challenges, in your browser.</h1>
      <p className="lead">
        CKA-style performance-based exercises. Each challenge spins up a real
        container, breaks the system, and gives you a terminal. Fix it, click{" "}
        <strong>Verify</strong>, move on.
      </p>
      <div className="cta">
        <Link to="/challenges" className="btn primary">
          Browse challenges →
        </Link>
        <Link to="/admin" className="btn ghost">
          Admin area
        </Link>
      </div>
      <div className="how">
        <div className="card">
          <h3>1. Pick a challenge</h3>
          <p>35+ exercises across Linux, networking, web, DB, docker, security, observability.</p>
        </div>
        <div className="card">
          <h3>2. Get a terminal</h3>
          <p>A disposable container with <code>bash + sudo</code> opens in the browser via ttyd.</p>
        </div>
        <div className="card">
          <h3>3. Verify</h3>
          <p>The platform runs a verification script in your container. Exit code 0 = pass.</p>
        </div>
      </div>
    </div>
  );
}
