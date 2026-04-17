import express from "express";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const VIDEOS_DIR = path.join(__dirname, "videos");
const THUMBS_DIR = path.join(__dirname, "thumbnails");
const PUBLIC_DIR = path.join(__dirname, "public");
const PORT = process.env.PORT || 3000;

for (const dir of [VIDEOS_DIR, THUMBS_DIR]) {
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
}

const VIDEO_EXT = new Set([".mp4", ".webm", ".mkv", ".mov", ".m4v"]);
const MIME = {
  ".mp4": "video/mp4",
  ".webm": "video/webm",
  ".mkv": "video/x-matroska",
  ".mov": "video/quicktime",
  ".m4v": "video/mp4",
};

const slugify = (s) =>
  s
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-zA-Z0-9._-]/g, "_");

const app = express();
app.use(express.static(PUBLIC_DIR));

app.get("/api/videos", (_req, res) => {
  const items = fs
    .readdirSync(VIDEOS_DIR)
    .filter((f) => VIDEO_EXT.has(path.extname(f).toLowerCase()))
    .map((f) => {
      const stat = fs.statSync(path.join(VIDEOS_DIR, f));
      const id = slugify(f);
      const thumbJpg = path.join(THUMBS_DIR, `${id}.jpg`);
      const thumbPng = path.join(THUMBS_DIR, `${id}.png`);
      const hasThumb = fs.existsSync(thumbJpg) || fs.existsSync(thumbPng);
      return {
        id,
        file: f,
        title: path.basename(f, path.extname(f)),
        size: stat.size,
        mtime: stat.mtimeMs,
        thumb: hasThumb ? `/thumbnails/${id}${fs.existsSync(thumbJpg) ? ".jpg" : ".png"}` : null,
        src: `/stream/${encodeURIComponent(f)}`,
      };
    })
    .sort((a, b) => b.mtime - a.mtime);
  res.json(items);
});

app.get("/thumbnails/:name", (req, res) => {
  const p = path.join(THUMBS_DIR, req.params.name);
  if (!p.startsWith(THUMBS_DIR) || !fs.existsSync(p)) return res.sendStatus(404);
  res.sendFile(p);
});

app.get("/stream/:name", (req, res) => {
  const name = decodeURIComponent(req.params.name);
  const filePath = path.join(VIDEOS_DIR, name);
  if (!filePath.startsWith(VIDEOS_DIR) || !fs.existsSync(filePath)) {
    return res.sendStatus(404);
  }

  const stat = fs.statSync(filePath);
  const total = stat.size;
  const ext = path.extname(filePath).toLowerCase();
  const contentType = MIME[ext] || "application/octet-stream";
  const range = req.headers.range;

  if (!range) {
    res.writeHead(200, {
      "Content-Length": total,
      "Content-Type": contentType,
      "Accept-Ranges": "bytes",
    });
    return fs.createReadStream(filePath).pipe(res);
  }

  const match = /bytes=(\d*)-(\d*)/.exec(range);
  const start = match[1] ? parseInt(match[1], 10) : 0;
  const end = match[2] ? parseInt(match[2], 10) : total - 1;

  if (start >= total || end >= total) {
    res.writeHead(416, { "Content-Range": `bytes */${total}` });
    return res.end();
  }

  res.writeHead(206, {
    "Content-Range": `bytes ${start}-${end}/${total}`,
    "Accept-Ranges": "bytes",
    "Content-Length": end - start + 1,
    "Content-Type": contentType,
  });

  fs.createReadStream(filePath, { start, end }).pipe(res);
});

app.listen(PORT, () => {
  console.log(`tryhard-player running → http://localhost:${PORT}`);
  console.log(`Drop videos into: ${VIDEOS_DIR}`);
});
