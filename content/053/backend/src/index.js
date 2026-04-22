import express from "express";
import cors from "cors";
import morgan from "morgan";
import { initDb, seedIfEmpty, ensureAdminUser } from "./db.js";
import { ensureBaseImage } from "./docker-manager.js";
import { authRouter } from "./routes/auth.js";
import { challengesRouter } from "./routes/challenges.js";
import { sessionsRouter } from "./routes/sessions.js";
import { adminRouter } from "./routes/admin.js";

const PORT = Number(process.env.PORT || 8054);

async function main() {
  initDb();
  seedIfEmpty();
  ensureAdminUser();

  // Best-effort: build the base image on boot. Don't block startup if it
  // fails — operator can rebuild manually.
  ensureBaseImage().catch((err) => {
    console.error("[boot] ensureBaseImage failed:", err.message);
  });

  const app = express();
  app.use(cors());
  app.use(express.json({ limit: "1mb" }));
  app.use(morgan("tiny"));

  app.get("/health", (_req, res) => res.json({ ok: true }));

  app.use("/api/auth", authRouter);
  app.use("/api/challenges", challengesRouter);
  app.use("/api/sessions", sessionsRouter);
  app.use("/api/admin", adminRouter);

  app.use((err, _req, res, _next) => {
    console.error(err);
    res.status(err.status || 500).json({ error: err.message || "internal" });
  });

  app.listen(PORT, "0.0.0.0", () => {
    console.log(`[backend] listening on :${PORT}`);
  });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
