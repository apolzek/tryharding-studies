import React, { useEffect, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { api } from "../api.js";

export default function ChallengePage() {
  const { slug } = useParams();
  const nav = useNavigate();
  const [challenge, setChallenge] = useState(null);
  const [session, setSession] = useState(null);
  const [status, setStatus] = useState("idle"); // idle | starting | running | verifying | passed | failed | stopping
  const [verifyResult, setVerifyResult] = useState(null);
  const [error, setError] = useState(null);
  const [remaining, setRemaining] = useState(null);
  const [showHints, setShowHints] = useState(false);
  const iframeRef = useRef(null);

  useEffect(() => {
    api.getChallenge(slug).then(setChallenge).catch((e) => setError(e.message));
  }, [slug]);

  // Countdown timer
  useEffect(() => {
    if (!session) return undefined;
    setRemaining(challenge.time_limit_sec);
    const id = setInterval(() => {
      setRemaining((r) => (r > 0 ? r - 1 : 0));
    }, 1000);
    return () => clearInterval(id);
  }, [session, challenge]);

  // Best-effort cleanup on unload.
  useEffect(() => {
    const onBeforeUnload = () => {
      if (session?.id) {
        navigator.sendBeacon?.(
          `${import.meta.env.VITE_API_URL || "http://localhost:8054"}/api/sessions/${session.id}/stop`
        );
      }
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [session]);

  async function startSession() {
    setError(null);
    setStatus("starting");
    try {
      const s = await api.startSession(slug);
      setSession(s);
      // wait a second for ttyd to bind
      setTimeout(() => setStatus("running"), 1500);
    } catch (e) {
      setError(e.message);
      setStatus("idle");
    }
  }

  async function verify() {
    if (!session) return;
    setStatus("verifying");
    setVerifyResult(null);
    try {
      const r = await api.verifySession(session.id);
      setVerifyResult(r);
      setStatus(r.passed ? "passed" : "failed");
    } catch (e) {
      setError(e.message);
      setStatus("running");
    }
  }

  async function stop() {
    if (!session) return;
    setStatus("stopping");
    try {
      await api.stopSession(session.id);
    } catch (e) {
      // ignore
    }
    setSession(null);
    setVerifyResult(null);
    setStatus("idle");
  }

  if (error) return <div className="error">Error: {error}</div>;
  if (!challenge) return <div className="loading">Loading…</div>;

  return (
    <div className="challenge-page">
      <div className="challenge-header">
        <button className="btn ghost" onClick={() => nav("/challenges")}>
          ← back
        </button>
        <div className="titles">
          <div className="breadcrumb">
            <span className={`tag tag-${challenge.difficulty}`}>{challenge.difficulty}</span>
            <span className="category">{challenge.category}</span>
          </div>
          <h1>{challenge.title}</h1>
        </div>
        <div className="timer">
          {remaining !== null && (
            <span className={remaining < 60 ? "low" : ""}>
              ⏱ {Math.floor(remaining / 60)}:
              {String(remaining % 60).padStart(2, "0")}
            </span>
          )}
        </div>
      </div>

      <div className="challenge-body">
        <aside className="objective-pane">
          <section>
            <h3>Objective</h3>
            <p>{challenge.objective}</p>
          </section>
          {challenge.description && (
            <section>
              <h3>Context</h3>
              <p>{challenge.description}</p>
            </section>
          )}
          {challenge.hints && challenge.hints.length > 0 && (
            <section>
              <h3>
                <button
                  className="link"
                  onClick={() => setShowHints((v) => !v)}
                >
                  Hints {showHints ? "▲" : "▼"}
                </button>
              </h3>
              {showHints && (
                <ul>
                  {challenge.hints.map((h, i) => (
                    <li key={i}>
                      <code>{h}</code>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          )}
          <section className="actions">
            {status === "idle" && (
              <button className="btn primary" onClick={startSession}>
                ▶ Start challenge
              </button>
            )}
            {status === "starting" && <div className="loading">Spawning container…</div>}
            {(status === "running" ||
              status === "verifying" ||
              status === "passed" ||
              status === "failed") && (
              <>
                <button
                  className="btn primary"
                  onClick={verify}
                  disabled={status === "verifying"}
                >
                  {status === "verifying" ? "Verifying…" : "✔ Verify"}
                </button>
                <button className="btn ghost" onClick={stop}>
                  ✕ Stop
                </button>
              </>
            )}
            {status === "stopping" && <div className="loading">Cleaning up…</div>}
          </section>
          {verifyResult && (
            <section
              className={`verify-result ${verifyResult.passed ? "pass" : "fail"}`}
            >
              <h3>{verifyResult.passed ? "✅ Passed" : "❌ Not yet"}</h3>
              <pre>{verifyResult.output || "(no output)"}</pre>
            </section>
          )}
        </aside>

        <section className="terminal-pane">
          {session ? (
            <iframe
              ref={iframeRef}
              src={session.terminal_url}
              title="challenge-terminal"
            />
          ) : (
            <div className="terminal-placeholder">
              <p>Click <strong>Start challenge</strong> to launch a container.</p>
              <p className="small">
                You'll get <code>bash</code> as <code>sre</code> user with passwordless
                sudo. The container is wiped when you stop the session.
              </p>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
