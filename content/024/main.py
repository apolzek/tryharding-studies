import json
import sqlite3
import asyncio
from contextlib import asynccontextmanager
from datetime import datetime, timezone

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.interval import IntervalTrigger

from github_client import fetch_repo_metadata

DB_PATH = "bot.db"
scheduler = AsyncIOScheduler()


def db():
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    return conn


def init_db():
    with db() as conn:
        conn.executescript(
            """
            CREATE TABLE IF NOT EXISTS repos (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                owner TEXT NOT NULL,
                repo TEXT NOT NULL,
                interval_minutes INTEGER NOT NULL DEFAULT 60,
                created_at TEXT NOT NULL,
                UNIQUE(owner, repo)
            );
            CREATE TABLE IF NOT EXISTS scans (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                repo_id INTEGER NOT NULL,
                scanned_at TEXT NOT NULL,
                ok INTEGER NOT NULL,
                error TEXT,
                metadata_json TEXT,
                FOREIGN KEY(repo_id) REFERENCES repos(id) ON DELETE CASCADE
            );
            """
        )


async def scan_repo(repo_id: int, owner: str, repo: str):
    now = datetime.now(timezone.utc).isoformat()
    try:
        meta = await fetch_repo_metadata(owner, repo)
        with db() as conn:
            conn.execute(
                "INSERT INTO scans (repo_id, scanned_at, ok, metadata_json) VALUES (?, ?, 1, ?)",
                (repo_id, now, json.dumps(meta)),
            )
        print(f"[scan] ok {owner}/{repo}")
    except Exception as e:
        with db() as conn:
            conn.execute(
                "INSERT INTO scans (repo_id, scanned_at, ok, error) VALUES (?, ?, 0, ?)",
                (repo_id, now, str(e)),
            )
        print(f"[scan] fail {owner}/{repo}: {e}")


def job_id(repo_id: int) -> str:
    return f"repo-{repo_id}"


def schedule_repo(repo_id: int, owner: str, repo: str, minutes: int):
    scheduler.add_job(
        scan_repo,
        trigger=IntervalTrigger(minutes=minutes),
        args=[repo_id, owner, repo],
        id=job_id(repo_id),
        replace_existing=True,
        next_run_time=datetime.now(timezone.utc),
    )


def load_jobs():
    with db() as conn:
        rows = conn.execute("SELECT * FROM repos").fetchall()
    for r in rows:
        schedule_repo(r["id"], r["owner"], r["repo"], r["interval_minutes"])


@asynccontextmanager
async def lifespan(app: FastAPI):
    init_db()
    scheduler.start()
    load_jobs()
    yield
    scheduler.shutdown()


app = FastAPI(title="GitHub Repo Pente-Fino Bot", lifespan=lifespan)


class RepoIn(BaseModel):
    owner: str = Field(..., min_length=1)
    repo: str = Field(..., min_length=1)
    interval_minutes: int = Field(60, ge=1)


@app.get("/")
def root():
    return {"service": "github-repo-pentefino", "status": "ok"}


@app.post("/repos", status_code=201)
def add_repo(body: RepoIn):
    now = datetime.now(timezone.utc).isoformat()
    try:
        with db() as conn:
            cur = conn.execute(
                "INSERT INTO repos (owner, repo, interval_minutes, created_at) VALUES (?, ?, ?, ?)",
                (body.owner, body.repo, body.interval_minutes, now),
            )
            repo_id = cur.lastrowid
    except sqlite3.IntegrityError:
        raise HTTPException(409, "repo already registered")
    schedule_repo(repo_id, body.owner, body.repo, body.interval_minutes)
    return {"id": repo_id, **body.model_dump()}


@app.get("/repos")
def list_repos():
    with db() as conn:
        return [dict(r) for r in conn.execute("SELECT * FROM repos").fetchall()]


@app.delete("/repos/{repo_id}")
def delete_repo(repo_id: int):
    with db() as conn:
        cur = conn.execute("DELETE FROM repos WHERE id = ?", (repo_id,))
        if cur.rowcount == 0:
            raise HTTPException(404, "not found")
    try:
        scheduler.remove_job(job_id(repo_id))
    except Exception:
        pass
    return {"deleted": repo_id}


@app.get("/repos/{repo_id}/scans")
def list_scans(repo_id: int, limit: int = 10):
    with db() as conn:
        rows = conn.execute(
            "SELECT id, scanned_at, ok, error, metadata_json FROM scans WHERE repo_id = ? ORDER BY id DESC LIMIT ?",
            (repo_id, limit),
        ).fetchall()
    out = []
    for r in rows:
        d = dict(r)
        if d.get("metadata_json"):
            d["metadata"] = json.loads(d.pop("metadata_json"))
        else:
            d.pop("metadata_json", None)
        out.append(d)
    return out


@app.get("/repos/{repo_id}/latest")
def latest_scan(repo_id: int):
    with db() as conn:
        row = conn.execute(
            "SELECT id, scanned_at, ok, error, metadata_json FROM scans WHERE repo_id = ? AND ok = 1 ORDER BY id DESC LIMIT 1",
            (repo_id,),
        ).fetchone()
    if not row:
        raise HTTPException(404, "no successful scan yet")
    d = dict(row)
    d["metadata"] = json.loads(d.pop("metadata_json"))
    return d


@app.post("/repos/{repo_id}/scan")
async def scan_now(repo_id: int):
    with db() as conn:
        row = conn.execute("SELECT * FROM repos WHERE id = ?", (repo_id,)).fetchone()
    if not row:
        raise HTTPException(404, "not found")
    await scan_repo(row["id"], row["owner"], row["repo"])
    return {"triggered": repo_id}
