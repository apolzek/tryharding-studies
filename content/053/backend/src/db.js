import Database from "better-sqlite3";
import bcrypt from "bcryptjs";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const DB_PATH = process.env.DB_PATH || "/data/sre.db";

let db;

export function getDb() {
  if (!db) throw new Error("db not initialized");
  return db;
}

export function initDb() {
  fs.mkdirSync(path.dirname(DB_PATH), { recursive: true });
  db = new Database(DB_PATH);
  db.pragma("journal_mode = WAL");
  db.exec(`
    CREATE TABLE IF NOT EXISTS users (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      username TEXT UNIQUE NOT NULL,
      password_hash TEXT NOT NULL,
      is_admin INTEGER NOT NULL DEFAULT 0,
      created_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS challenges (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      slug TEXT UNIQUE NOT NULL,
      title TEXT NOT NULL,
      category TEXT NOT NULL,
      difficulty TEXT NOT NULL,
      time_limit_sec INTEGER NOT NULL DEFAULT 900,
      objective TEXT NOT NULL,
      description TEXT NOT NULL DEFAULT '',
      hints TEXT NOT NULL DEFAULT '[]',
      setup_script TEXT NOT NULL,
      verify_script TEXT NOT NULL,
      privileged INTEGER NOT NULL DEFAULT 0,
      enabled INTEGER NOT NULL DEFAULT 1,
      created_at TEXT NOT NULL DEFAULT (datetime('now')),
      updated_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS sessions (
      id TEXT PRIMARY KEY,
      challenge_id INTEGER NOT NULL,
      container_id TEXT,
      port INTEGER,
      status TEXT NOT NULL DEFAULT 'starting',
      passed INTEGER NOT NULL DEFAULT 0,
      verify_output TEXT DEFAULT '',
      started_at TEXT NOT NULL DEFAULT (datetime('now')),
      completed_at TEXT,
      FOREIGN KEY (challenge_id) REFERENCES challenges(id)
    );

    CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
    CREATE INDEX IF NOT EXISTS idx_challenges_enabled ON challenges(enabled);
  `);
}

export function seedIfEmpty() {
  const row = db.prepare("SELECT COUNT(*) AS c FROM challenges").get();
  if (row.c > 0) return;
  const seedPath = path.join(__dirname, "..", "seed", "challenges.json");
  const seed = JSON.parse(fs.readFileSync(seedPath, "utf8"));
  const insert = db.prepare(`
    INSERT INTO challenges
      (slug, title, category, difficulty, time_limit_sec, objective,
       description, hints, setup_script, verify_script, privileged, enabled)
    VALUES (@slug, @title, @category, @difficulty, @time_limit_sec, @objective,
            @description, @hints, @setup_script, @verify_script, @privileged, 1)
  `);
  const tx = db.transaction((items) => {
    for (const c of items) {
      insert.run({
        slug: c.slug,
        title: c.title,
        category: c.category,
        difficulty: c.difficulty,
        time_limit_sec: c.time_limit_sec ?? 900,
        objective: c.objective,
        description: c.description || "",
        hints: JSON.stringify(c.hints || []),
        setup_script: c.setup_script,
        verify_script: c.verify_script,
        privileged: c.privileged ? 1 : 0,
      });
    }
  });
  tx(seed);
  console.log(`[seed] inserted ${seed.length} challenges`);
}

export function ensureAdminUser() {
  const username = process.env.ADMIN_USER || "admin";
  const password = process.env.ADMIN_PASSWORD || "admin123";
  const row = db.prepare("SELECT id FROM users WHERE username = ?").get(username);
  if (row) return;
  const hash = bcrypt.hashSync(password, 10);
  db.prepare(
    "INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, 1)"
  ).run(username, hash);
  console.log(`[seed] created admin user: ${username}`);
}
