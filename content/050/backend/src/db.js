const path = require('path');
const fs = require('fs');
const Database = require('better-sqlite3');

function openDb(dbPath) {
  if (dbPath !== ':memory:') {
    const dir = path.dirname(dbPath);
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
  }
  const db = new Database(dbPath);
  db.pragma('journal_mode = WAL');
  db.pragma('foreign_keys = ON');
  return db;
}

function migrate(db) {
  db.exec(`
    CREATE TABLE IF NOT EXISTS users (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      username TEXT UNIQUE NOT NULL,
      password_hash TEXT NOT NULL,
      display_name TEXT NOT NULL,
      photo_url TEXT DEFAULT '',
      bio TEXT DEFAULT '',
      status TEXT DEFAULT 'solteiro(a)',
      age INTEGER DEFAULT 0,
      city TEXT DEFAULT '',
      country TEXT DEFAULT 'Brasil',
      created_at INTEGER NOT NULL
    );

    CREATE TABLE IF NOT EXISTS scraps (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      profile_user_id INTEGER NOT NULL,
      author_user_id INTEGER NOT NULL,
      body TEXT NOT NULL,
      created_at INTEGER NOT NULL,
      FOREIGN KEY(profile_user_id) REFERENCES users(id) ON DELETE CASCADE,
      FOREIGN KEY(author_user_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS friendships (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      requester_id INTEGER NOT NULL,
      addressee_id INTEGER NOT NULL,
      status TEXT NOT NULL CHECK(status IN ('pending','accepted')),
      created_at INTEGER NOT NULL,
      UNIQUE(requester_id, addressee_id),
      FOREIGN KEY(requester_id) REFERENCES users(id) ON DELETE CASCADE,
      FOREIGN KEY(addressee_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS communities (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      owner_id INTEGER NOT NULL,
      name TEXT NOT NULL,
      description TEXT DEFAULT '',
      category TEXT DEFAULT 'Geral',
      photo_url TEXT DEFAULT '',
      created_at INTEGER NOT NULL,
      FOREIGN KEY(owner_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS community_members (
      community_id INTEGER NOT NULL,
      user_id INTEGER NOT NULL,
      joined_at INTEGER NOT NULL,
      PRIMARY KEY(community_id, user_id),
      FOREIGN KEY(community_id) REFERENCES communities(id) ON DELETE CASCADE,
      FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS testimonials (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      profile_user_id INTEGER NOT NULL,
      author_user_id INTEGER NOT NULL,
      body TEXT NOT NULL,
      created_at INTEGER NOT NULL,
      FOREIGN KEY(profile_user_id) REFERENCES users(id) ON DELETE CASCADE,
      FOREIGN KEY(author_user_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS ratings (
      rater_id INTEGER NOT NULL,
      ratee_id INTEGER NOT NULL,
      trust INTEGER DEFAULT 0 CHECK(trust BETWEEN 0 AND 3),
      cool INTEGER DEFAULT 0 CHECK(cool BETWEEN 0 AND 3),
      sexy INTEGER DEFAULT 0 CHECK(sexy BETWEEN 0 AND 3),
      is_fan INTEGER DEFAULT 0 CHECK(is_fan IN (0,1)),
      updated_at INTEGER NOT NULL,
      PRIMARY KEY(rater_id, ratee_id),
      FOREIGN KEY(rater_id) REFERENCES users(id) ON DELETE CASCADE,
      FOREIGN KEY(ratee_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS visits (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      profile_user_id INTEGER NOT NULL,
      visitor_user_id INTEGER NOT NULL,
      visited_at INTEGER NOT NULL,
      FOREIGN KEY(profile_user_id) REFERENCES users(id) ON DELETE CASCADE,
      FOREIGN KEY(visitor_user_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE TABLE IF NOT EXISTS photos (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      user_id INTEGER NOT NULL,
      url TEXT NOT NULL,
      caption TEXT DEFAULT '',
      created_at INTEGER NOT NULL,
      FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
    );
  `);
}

module.exports = { openDb, migrate };
