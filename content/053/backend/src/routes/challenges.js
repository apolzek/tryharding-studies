import { Router } from "express";
import { getDb } from "../db.js";

export const challengesRouter = Router();

// Public listing — no scripts, no verify content (avoid leaking answers).
challengesRouter.get("/", (_req, res) => {
  const rows = getDb()
    .prepare(
      `SELECT id, slug, title, category, difficulty, time_limit_sec, objective, hints
         FROM challenges
        WHERE enabled = 1
        ORDER BY category, difficulty, title`
    )
    .all();
  res.json(
    rows.map((r) => ({
      ...r,
      hints: JSON.parse(r.hints || "[]"),
    }))
  );
});

challengesRouter.get("/:slug", (req, res) => {
  const row = getDb()
    .prepare(
      `SELECT id, slug, title, category, difficulty, time_limit_sec, objective,
              description, hints
         FROM challenges
        WHERE slug = ? AND enabled = 1`
    )
    .get(req.params.slug);
  if (!row) return res.status(404).json({ error: "not found" });
  row.hints = JSON.parse(row.hints || "[]");
  res.json(row);
});
