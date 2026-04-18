---
title: tryhard-player — local YouTube-style video player from scratch
tags: [frontend, video, nodejs, http-range-streaming, ffmpeg]
status: stable
---

# tryhard-player

Local YouTube-style video player, from scratch. Reusable engine + library UI.

## How to run

```bash
cd content/038
npm install
npm start
```

Open `http://localhost:3000`. Drop files into `videos/` (`.mp4`, `.webm`, `.mkv`, `.mov`, `.m4v`).

### Thumbnails (optional)

Thumbnail generation is out-of-engine — just place `thumbnails/<id>.jpg` where `<id>` is the slugified video filename. To batch-generate with ffmpeg:

```bash
for f in videos/*.mp4; do
  name=$(basename "$f" | tr -cs 'A-Za-z0-9._-' '_')
  ffmpeg -ss 00:00:02 -i "$f" -vframes 1 -q:v 3 "thumbnails/${name}.jpg"
done
```

Without a thumbnail, the grid uses the video itself at `#t=2` as preview.

## Engine (`public/js/player.js`)

`TryhardPlayer` class, zero dependencies. To embed it in another project:

```html
<link rel="stylesheet" href="player.css" />
<div id="p"></div>
<script type="module">
  import { TryhardPlayer } from "./player.js";
  const player = new TryhardPlayer(document.getElementById("p"), {
    src: "/stream/video.mp4",
    title: "My video",
    autoplay: false,
  });
</script>
```

API:
- `load({ src, poster, title })`
- `play()`, `pause()`, `toggle()`
- `toggleMute()`, `toggleFullscreen()`, `togglePip()`
- `destroy()`

Keyboard shortcuts: `space`/`k` play, `m` mute, `f` fullscreen, `i` PiP, `j`/`l` ±10s, `←`/`→` ±5s, `↑`/`↓` volume, `0–9` percentage seek.

## Backend (`server.js`)

- `GET /api/videos` — lists files under `videos/`
- `GET /stream/:name` — HTTP Range streaming (fast seek, no full-file buffering in RAM)
- `GET /thumbnails/:name` — static thumbnails
