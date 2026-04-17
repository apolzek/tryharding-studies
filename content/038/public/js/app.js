import { TryhardPlayer } from "./player.js";

const app = document.getElementById("app");
const search = document.getElementById("search");

const fmtSize = (b) => {
  if (b < 1024) return b + " B";
  if (b < 1024 ** 2) return (b / 1024).toFixed(1) + " KB";
  if (b < 1024 ** 3) return (b / 1024 ** 2).toFixed(1) + " MB";
  return (b / 1024 ** 3).toFixed(2) + " GB";
};
const fmtDate = (ms) => new Date(ms).toLocaleDateString("pt-BR");

let ALL = [];
let player = null;
let query = "";

async function loadVideos() {
  const r = await fetch("/api/videos");
  ALL = await r.json();
}

function thumb(v, tag = "img") {
  if (v.thumb) return `<${tag} src="${v.thumb}" loading="lazy">`;
  return `<video src="${v.src}#t=2" preload="metadata" muted></video>`;
}

function renderHome() {
  if (player) { player.destroy(); player = null; }
  const filtered = ALL.filter((v) =>
    v.title.toLowerCase().includes(query.toLowerCase())
  );

  if (ALL.length === 0) {
    app.innerHTML = `
      <div class="empty">
        <h2>Nenhum vídeo ainda</h2>
        <p>Adicione arquivos em <code>videos/</code> e recarregue.</p>
        <p class="muted">Formatos suportados: mp4, webm, mkv, mov, m4v</p>
      </div>`;
    return;
  }
  if (filtered.length === 0) {
    app.innerHTML = `<div class="empty"><h2>Nada encontrado</h2><p>Tente outro termo.</p></div>`;
    return;
  }

  app.innerHTML = `
    <div class="grid">
      ${filtered
        .map(
          (v) => `
        <a class="card" href="#/watch/${encodeURIComponent(v.id)}">
          <div class="card-thumb">
            ${thumb(v)}
            <span class="card-size">${fmtSize(v.size)}</span>
          </div>
          <div class="card-info">
            <div class="card-title">${v.title}</div>
            <div class="card-meta">${fmtDate(v.mtime)}</div>
          </div>
        </a>`
        )
        .join("")}
    </div>`;
}

function renderWatch(id) {
  const v = ALL.find((x) => x.id === id);
  if (!v) { location.hash = "#/"; return; }
  const others = ALL.filter((x) => x.id !== id).slice(0, 20);

  app.innerHTML = `
    <div class="watch">
      <div class="watch-player">
        <a class="watch-back" href="#/">← voltar</a>
        <div id="player"></div>
        <h1 class="watch-title">${v.title}</h1>
        <div class="watch-meta">${fmtSize(v.size)} · ${fmtDate(v.mtime)}</div>
      </div>
      <aside class="sidebar">
        <h3>Mais vídeos</h3>
        ${others
          .map(
            (o) => `
          <a class="side-card" href="#/watch/${encodeURIComponent(o.id)}">
            <div class="side-thumb">${thumb(o)}</div>
            <div class="side-info">
              <div class="t">${o.title}</div>
              <div class="m">${fmtSize(o.size)} · ${fmtDate(o.mtime)}</div>
            </div>
          </a>`
          )
          .join("")}
      </aside>
    </div>`;

  if (player) player.destroy();
  player = new TryhardPlayer(document.getElementById("player"), {
    src: v.src,
    poster: v.thumb || "",
    title: v.title,
    autoplay: true,
  });
}

function route() {
  const h = location.hash || "#/";
  const m = h.match(/^#\/watch\/(.+)$/);
  if (m) renderWatch(decodeURIComponent(m[1]));
  else renderHome();
}

search.addEventListener("input", (e) => {
  query = e.target.value;
  if (!location.hash || location.hash === "#/") renderHome();
  else location.hash = "#/";
});

window.addEventListener("hashchange", route);

(async () => {
  await loadVideos();
  route();
})();
