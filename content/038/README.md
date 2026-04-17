# tryhard-player

Player de vídeo local estilo YouTube, do zero. Engine reutilizável + UI de biblioteca.

## Como rodar

```bash
cd content/038
npm install
npm start
```

Abra `http://localhost:3000`. Coloque arquivos em `videos/` (`.mp4`, `.webm`, `.mkv`, `.mov`, `.m4v`).

### Thumbnails (opcional)

Geração fora da engine — basta colocar `thumbnails/<id>.jpg` onde `<id>` é o nome do arquivo de vídeo slugificado. Para gerar em massa com ffmpeg:

```bash
for f in videos/*.mp4; do
  name=$(basename "$f" | tr -cs 'A-Za-z0-9._-' '_')
  ffmpeg -ss 00:00:02 -i "$f" -vframes 1 -q:v 3 "thumbnails/${name}.jpg"
done
```

Sem thumbnail, o grid usa o próprio vídeo em `#t=2` como preview.

## Engine (`public/js/player.js`)

Classe `TryhardPlayer`, zero dependências. Para usar em outro projeto:

```html
<link rel="stylesheet" href="player.css" />
<div id="p"></div>
<script type="module">
  import { TryhardPlayer } from "./player.js";
  const player = new TryhardPlayer(document.getElementById("p"), {
    src: "/stream/video.mp4",
    title: "Meu vídeo",
    autoplay: false,
  });
</script>
```

API:
- `load({ src, poster, title })`
- `play()`, `pause()`, `toggle()`
- `toggleMute()`, `toggleFullscreen()`, `togglePip()`
- `destroy()`

Atalhos de teclado: `space`/`k` play, `m` mute, `f` fullscreen, `i` PiP, `j`/`l` ±10s, `←`/`→` ±5s, `↑`/`↓` volume, `0–9` seek percentual.

## Backend (`server.js`)

- `GET /api/videos` — lista os arquivos em `videos/`
- `GET /stream/:name` — streaming com HTTP Range (seek rápido, sem carregar tudo na RAM)
- `GET /thumbnails/:name` — thumbnails estáticas
