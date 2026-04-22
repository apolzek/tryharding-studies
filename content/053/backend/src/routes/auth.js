import { Router } from "express";
import bcrypt from "bcryptjs";
import { getDb } from "../db.js";
import { signToken } from "../auth.js";

export const authRouter = Router();

authRouter.post("/login", (req, res) => {
  const { username, password } = req.body || {};
  if (!username || !password) {
    return res.status(400).json({ error: "username and password required" });
  }
  const row = getDb()
    .prepare("SELECT id, username, password_hash, is_admin FROM users WHERE username = ?")
    .get(username);
  if (!row || !bcrypt.compareSync(password, row.password_hash)) {
    return res.status(401).json({ error: "invalid credentials" });
  }
  const token = signToken({
    sub: row.id,
    username: row.username,
    is_admin: !!row.is_admin,
  });
  res.json({ token, user: { id: row.id, username: row.username, is_admin: !!row.is_admin } });
});
