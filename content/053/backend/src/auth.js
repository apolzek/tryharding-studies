import jwt from "jsonwebtoken";

const SECRET = process.env.JWT_SECRET || "please-change-me";

export function signToken(payload) {
  return jwt.sign(payload, SECRET, { expiresIn: "12h" });
}

export function verifyToken(token) {
  return jwt.verify(token, SECRET);
}

export function requireAdmin(req, res, next) {
  const auth = req.headers.authorization || "";
  const token = auth.startsWith("Bearer ") ? auth.slice(7) : null;
  if (!token) return res.status(401).json({ error: "missing token" });
  try {
    const claims = verifyToken(token);
    if (!claims.is_admin) return res.status(403).json({ error: "admin only" });
    req.user = claims;
    next();
  } catch {
    res.status(401).json({ error: "invalid token" });
  }
}
