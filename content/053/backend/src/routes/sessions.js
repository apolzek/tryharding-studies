import { Router } from "express";
import { randomUUID } from "node:crypto";
import { getDb } from "../db.js";
import {
  allocatePort,
  releasePort,
  startChallengeContainer,
  stopChallengeContainer,
  execVerify,
} from "../docker-manager.js";

export const sessionsRouter = Router();

const CHALLENGE_HOST = process.env.CHALLENGE_HOST || "localhost";

// POST /api/sessions  { slug }
// Spawns a container for the challenge and returns its URL.
sessionsRouter.post("/", async (req, res, next) => {
  try {
    const { slug } = req.body || {};
    if (!slug) return res.status(400).json({ error: "slug required" });

    const challenge = getDb()
      .prepare("SELECT * FROM challenges WHERE slug = ? AND enabled = 1")
      .get(slug);
    if (!challenge) return res.status(404).json({ error: "challenge not found" });

    const port = allocatePort();
    const sessionId = randomUUID();
    const db = getDb();
    db.prepare(
      `INSERT INTO sessions (id, challenge_id, port, status)
       VALUES (?, ?, ?, 'starting')`
    ).run(sessionId, challenge.id, port);

    let container;
    try {
      container = await startChallengeContainer(challenge, port);
    } catch (err) {
      releasePort(port);
      db.prepare("UPDATE sessions SET status='failed' WHERE id = ?").run(sessionId);
      throw err;
    }

    db.prepare(
      `UPDATE sessions SET container_id = ?, status='running' WHERE id = ?`
    ).run(container.id, sessionId);

    res.json({
      id: sessionId,
      port,
      terminal_url: `http://${CHALLENGE_HOST}:${port}`,
      challenge: {
        slug: challenge.slug,
        title: challenge.title,
        objective: challenge.objective,
        time_limit_sec: challenge.time_limit_sec,
      },
    });
  } catch (err) {
    next(err);
  }
});

sessionsRouter.get("/:id", (req, res) => {
  const row = getDb().prepare("SELECT * FROM sessions WHERE id = ?").get(req.params.id);
  if (!row) return res.status(404).json({ error: "not found" });
  res.json(row);
});

// POST /api/sessions/:id/verify  → run verify.sh
sessionsRouter.post("/:id/verify", async (req, res, next) => {
  try {
    const db = getDb();
    const session = db.prepare("SELECT * FROM sessions WHERE id = ?").get(req.params.id);
    if (!session) return res.status(404).json({ error: "not found" });
    if (session.status !== "running") {
      return res.status(409).json({ error: `session is ${session.status}` });
    }
    const { exitCode, output } = await execVerify(session.container_id);
    const passed = exitCode === 0 ? 1 : 0;
    db.prepare(
      `UPDATE sessions SET passed = ?, verify_output = ? WHERE id = ?`
    ).run(passed, output, session.id);
    res.json({ passed: !!passed, exit_code: exitCode, output });
  } catch (err) {
    next(err);
  }
});

// POST /api/sessions/:id/stop
sessionsRouter.post("/:id/stop", async (req, res, next) => {
  try {
    const db = getDb();
    const session = db.prepare("SELECT * FROM sessions WHERE id = ?").get(req.params.id);
    if (!session) return res.status(404).json({ error: "not found" });
    if (session.container_id) await stopChallengeContainer(session.container_id);
    if (session.port) releasePort(session.port);
    db.prepare(
      `UPDATE sessions SET status='stopped', completed_at = datetime('now') WHERE id = ?`
    ).run(session.id);
    res.json({ stopped: true });
  } catch (err) {
    next(err);
  }
});
