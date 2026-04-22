import { Router } from "express";
import { getDb } from "../db.js";
import { requireAdmin } from "../auth.js";

export const adminRouter = Router();

adminRouter.use(requireAdmin);

// List challenges (full records, including scripts)
adminRouter.get("/challenges", (_req, res) => {
  const rows = getDb()
    .prepare(
      `SELECT id, slug, title, category, difficulty, time_limit_sec, objective,
              description, hints, setup_script, verify_script, privileged, enabled,
              created_at, updated_at
         FROM challenges ORDER BY id DESC`
    )
    .all();
  res.json(rows.map((r) => ({ ...r, hints: JSON.parse(r.hints || "[]") })));
});

adminRouter.get("/challenges/:id", (req, res) => {
  const row = getDb().prepare("SELECT * FROM challenges WHERE id = ?").get(req.params.id);
  if (!row) return res.status(404).json({ error: "not found" });
  row.hints = JSON.parse(row.hints || "[]");
  res.json(row);
});

adminRouter.post("/challenges", (req, res) => {
  const b = req.body || {};
  if (!b.slug || !b.title || !b.objective || !b.setup_script || !b.verify_script) {
    return res.status(400).json({ error: "slug/title/objective/setup_script/verify_script required" });
  }
  const db = getDb();
  const existing = db.prepare("SELECT id FROM challenges WHERE slug = ?").get(b.slug);
  if (existing) return res.status(409).json({ error: "slug already exists" });
  const info = db
    .prepare(
      `INSERT INTO challenges
         (slug, title, category, difficulty, time_limit_sec, objective, description,
          hints, setup_script, verify_script, privileged, enabled)
       VALUES (@slug, @title, @category, @difficulty, @time_limit_sec, @objective,
               @description, @hints, @setup_script, @verify_script, @privileged, @enabled)`
    )
    .run({
      slug: b.slug,
      title: b.title,
      category: b.category || "linux",
      difficulty: b.difficulty || "medium",
      time_limit_sec: b.time_limit_sec ?? 900,
      objective: b.objective,
      description: b.description || "",
      hints: JSON.stringify(b.hints || []),
      setup_script: b.setup_script,
      verify_script: b.verify_script,
      privileged: b.privileged ? 1 : 0,
      enabled: b.enabled === false ? 0 : 1,
    });
  res.status(201).json({ id: info.lastInsertRowid });
});

adminRouter.put("/challenges/:id", (req, res) => {
  const b = req.body || {};
  const db = getDb();
  const existing = db.prepare("SELECT id FROM challenges WHERE id = ?").get(req.params.id);
  if (!existing) return res.status(404).json({ error: "not found" });
  db.prepare(
    `UPDATE challenges SET
       slug = COALESCE(@slug, slug),
       title = COALESCE(@title, title),
       category = COALESCE(@category, category),
       difficulty = COALESCE(@difficulty, difficulty),
       time_limit_sec = COALESCE(@time_limit_sec, time_limit_sec),
       objective = COALESCE(@objective, objective),
       description = COALESCE(@description, description),
       hints = COALESCE(@hints, hints),
       setup_script = COALESCE(@setup_script, setup_script),
       verify_script = COALESCE(@verify_script, verify_script),
       privileged = COALESCE(@privileged, privileged),
       enabled = COALESCE(@enabled, enabled),
       updated_at = datetime('now')
     WHERE id = @id`
  ).run({
    id: req.params.id,
    slug: b.slug ?? null,
    title: b.title ?? null,
    category: b.category ?? null,
    difficulty: b.difficulty ?? null,
    time_limit_sec: b.time_limit_sec ?? null,
    objective: b.objective ?? null,
    description: b.description ?? null,
    hints: b.hints ? JSON.stringify(b.hints) : null,
    setup_script: b.setup_script ?? null,
    verify_script: b.verify_script ?? null,
    privileged:
      b.privileged === undefined ? null : b.privileged ? 1 : 0,
    enabled: b.enabled === undefined ? null : b.enabled ? 1 : 0,
  });
  res.json({ updated: true });
});

adminRouter.delete("/challenges/:id", (req, res) => {
  const info = getDb().prepare("DELETE FROM challenges WHERE id = ?").run(req.params.id);
  if (info.changes === 0) return res.status(404).json({ error: "not found" });
  res.json({ deleted: true });
});

adminRouter.get("/sessions", (_req, res) => {
  const rows = getDb()
    .prepare(
      `SELECT s.*, c.slug, c.title
         FROM sessions s
         JOIN challenges c ON c.id = s.challenge_id
         ORDER BY s.started_at DESC
         LIMIT 200`
    )
    .all();
  res.json(rows);
});
