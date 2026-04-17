// TryhardPlayer — engine de vídeo estilo YouTube, sem dependências.
// Uso:
//   const p = new TryhardPlayer(containerEl, { src, poster, autoplay });
//   p.load({ src, poster, title });
//   p.play(); p.pause(); p.destroy();

const fmtTime = (s) => {
  if (!isFinite(s) || s < 0) s = 0;
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = Math.floor(s % 60).toString().padStart(2, "0");
  return h > 0 ? `${h}:${m.toString().padStart(2, "0")}:${sec}` : `${m}:${sec}`;
};

const SVG = {
  play: '<svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>',
  pause: '<svg viewBox="0 0 24 24"><path d="M6 5h4v14H6zM14 5h4v14h-4z"/></svg>',
  volHigh: '<svg viewBox="0 0 24 24"><path d="M3 10v4h4l5 5V5L7 10H3zm13.5 2a4.5 4.5 0 0 0-2.5-4v8a4.5 4.5 0 0 0 2.5-4zM14 3.2v2.1a7 7 0 0 1 0 13.4v2.1a9 9 0 0 0 0-17.6z"/></svg>',
  volMute: '<svg viewBox="0 0 24 24"><path d="M16.5 12a4.5 4.5 0 0 0-2.5-4v2.2l2.5 2.5zM19 12c0 .9-.2 1.8-.5 2.6l1.5 1.5A9 9 0 0 0 21 12a9 9 0 0 0-7-8.8v2.1a7 7 0 0 1 5 6.7zM4.3 3L3 4.3 7.7 9H3v6h4l5 5v-6.7l4.2 4.2c-.7.5-1.4.9-2.2 1.1v2.1c1.4-.3 2.6-.9 3.7-1.7L19.7 21l1.3-1.3L4.3 3zM12 4L9.9 6.1 12 8.2V4z"/></svg>',
  fsEnter: '<svg viewBox="0 0 24 24"><path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/></svg>',
  fsExit: '<svg viewBox="0 0 24 24"><path d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-11V5h-2v5h5V8h-3z"/></svg>',
  settings: '<svg viewBox="0 0 24 24"><path d="M19.4 13c.04-.3.06-.6.06-1s-.02-.7-.06-1l2.1-1.6a.5.5 0 0 0 .1-.6l-2-3.4a.5.5 0 0 0-.6-.2l-2.4 1a7 7 0 0 0-1.8-1L14.5 2.5a.5.5 0 0 0-.5-.4h-4a.5.5 0 0 0-.5.4L9.2 5a7 7 0 0 0-1.8 1L5 5a.5.5 0 0 0-.6.2l-2 3.4a.5.5 0 0 0 .1.6L4.6 11c-.04.3-.06.6-.06 1s.02.7.06 1L2.5 14.6a.5.5 0 0 0-.1.6l2 3.4c.14.2.4.3.6.2l2.4-1a7 7 0 0 0 1.8 1l.3 2.5c.05.25.26.4.5.4h4c.24 0 .45-.15.5-.4l.3-2.5a7 7 0 0 0 1.8-1l2.4 1c.23.1.48 0 .6-.2l2-3.4a.5.5 0 0 0-.1-.6L19.4 13zM12 15.5a3.5 3.5 0 1 1 0-7 3.5 3.5 0 0 1 0 7z"/></svg>',
  pip: '<svg viewBox="0 0 24 24"><path d="M19 11h-8v6h8v-6zm4 8V4.98C23 3.88 22.1 3 21 3H3c-1.1 0-2 .88-2 1.98V19c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2zm-2 .02H3V4.97h18v14.05z"/></svg>',
};

export class TryhardPlayer {
  constructor(container, opts = {}) {
    this.container = container;
    this.opts = { autoplay: false, ...opts };
    this.lastVolume = 1;
    this._build();
    this._bind();
    if (opts.src) this.load(opts);
  }

  _build() {
    this.container.classList.add("thp");
    this.container.innerHTML = `
      <video class="thp-video" playsinline preload="metadata"></video>
      <div class="thp-title"></div>
      <div class="thp-center"></div>
      <div class="thp-controls">
        <div class="thp-progress">
          <div class="thp-progress-buffer"></div>
          <div class="thp-progress-played"></div>
          <div class="thp-progress-hover"></div>
          <div class="thp-progress-thumb"></div>
          <div class="thp-progress-tooltip">0:00</div>
        </div>
        <div class="thp-bar">
          <button class="thp-btn thp-play" title="Play (k)">${SVG.play}</button>
          <div class="thp-volume">
            <button class="thp-btn thp-mute" title="Mute (m)">${SVG.volHigh}</button>
            <input class="thp-vol-slider" type="range" min="0" max="1" step="0.01" value="1">
          </div>
          <div class="thp-time"><span class="thp-cur">0:00</span> / <span class="thp-dur">0:00</span></div>
          <div class="thp-spacer"></div>
          <div class="thp-menu">
            <button class="thp-btn thp-speed" title="Playback speed">1x</button>
            <div class="thp-speed-menu">
              ${[0.25, 0.5, 0.75, 1, 1.25, 1.5, 2].map((s) => `<div data-speed="${s}"${s === 1 ? ' class="active"' : ""}>${s}x</div>`).join("")}
            </div>
          </div>
          <button class="thp-btn thp-pip" title="Picture-in-picture (i)">${SVG.pip}</button>
          <button class="thp-btn thp-fs" title="Fullscreen (f)">${SVG.fsEnter}</button>
        </div>
      </div>
    `;
    this.video = this.container.querySelector(".thp-video");
    this.titleEl = this.container.querySelector(".thp-title");
    this.centerEl = this.container.querySelector(".thp-center");
    this.progress = this.container.querySelector(".thp-progress");
    this.playedEl = this.container.querySelector(".thp-progress-played");
    this.bufferEl = this.container.querySelector(".thp-progress-buffer");
    this.hoverEl = this.container.querySelector(".thp-progress-hover");
    this.thumbEl = this.container.querySelector(".thp-progress-thumb");
    this.tooltipEl = this.container.querySelector(".thp-progress-tooltip");
    this.playBtn = this.container.querySelector(".thp-play");
    this.muteBtn = this.container.querySelector(".thp-mute");
    this.volSlider = this.container.querySelector(".thp-vol-slider");
    this.curEl = this.container.querySelector(".thp-cur");
    this.durEl = this.container.querySelector(".thp-dur");
    this.speedBtn = this.container.querySelector(".thp-speed");
    this.speedMenu = this.container.querySelector(".thp-speed-menu");
    this.pipBtn = this.container.querySelector(".thp-pip");
    this.fsBtn = this.container.querySelector(".thp-fs");
  }

  _bind() {
    const v = this.video;

    this.container.addEventListener("click", (e) => {
      if (e.target === this.container || e.target === v || e.target === this.centerEl) {
        this.toggle();
      }
    });
    this.container.addEventListener("dblclick", (e) => {
      if (e.target === v || e.target === this.centerEl) this.toggleFullscreen();
    });

    this.playBtn.addEventListener("click", () => this.toggle());
    v.addEventListener("play", () => {
      this.playBtn.innerHTML = SVG.pause;
      this.container.classList.add("playing");
      this.container.classList.remove("paused");
    });
    v.addEventListener("pause", () => {
      this.playBtn.innerHTML = SVG.play;
      this.container.classList.remove("playing");
      this.container.classList.add("paused");
    });
    v.addEventListener("ended", () => {
      this.container.classList.remove("playing");
      this.container.classList.add("paused");
    });

    v.addEventListener("loadedmetadata", () => {
      this.durEl.textContent = fmtTime(v.duration);
    });
    v.addEventListener("timeupdate", () => {
      const pct = (v.currentTime / v.duration) * 100 || 0;
      this.playedEl.style.width = pct + "%";
      this.thumbEl.style.left = pct + "%";
      this.curEl.textContent = fmtTime(v.currentTime);
    });
    v.addEventListener("progress", () => {
      if (v.buffered.length) {
        const end = v.buffered.end(v.buffered.length - 1);
        this.bufferEl.style.width = ((end / v.duration) * 100 || 0) + "%";
      }
    });
    v.addEventListener("waiting", () => this.container.classList.add("loading"));
    v.addEventListener("canplay", () => this.container.classList.remove("loading"));

    // progress bar
    const seekFromEvent = (e) => {
      const rect = this.progress.getBoundingClientRect();
      const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width));
      return { pct: x / rect.width, x };
    };
    this.progress.addEventListener("mousemove", (e) => {
      const { pct, x } = seekFromEvent(e);
      this.hoverEl.style.width = pct * 100 + "%";
      this.tooltipEl.textContent = fmtTime(pct * (v.duration || 0));
      this.tooltipEl.style.left = x + "px";
    });
    this.progress.addEventListener("mouseleave", () => {
      this.hoverEl.style.width = "0%";
    });
    let dragging = false;
    const onDown = (e) => {
      dragging = true;
      const { pct } = seekFromEvent(e);
      v.currentTime = pct * v.duration;
    };
    const onMove = (e) => {
      if (!dragging) return;
      const { pct } = seekFromEvent(e);
      v.currentTime = pct * v.duration;
    };
    const onUp = () => (dragging = false);
    this.progress.addEventListener("mousedown", onDown);
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    this._cleanup = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };

    // volume
    this.muteBtn.addEventListener("click", () => this.toggleMute());
    this.volSlider.addEventListener("input", (e) => {
      v.volume = parseFloat(e.target.value);
      v.muted = v.volume === 0;
      this._updateVolIcon();
    });
    v.addEventListener("volumechange", () => {
      this.volSlider.value = v.muted ? 0 : v.volume;
      this._updateVolIcon();
    });

    // speed
    this.speedBtn.addEventListener("click", (e) => {
      e.stopPropagation();
      this.speedMenu.classList.toggle("open");
    });
    this.speedMenu.addEventListener("click", (e) => {
      const s = e.target.dataset.speed;
      if (!s) return;
      v.playbackRate = parseFloat(s);
      this.speedBtn.textContent = s === "1" ? "1x" : `${s}x`;
      this.speedMenu.querySelectorAll("div").forEach((d) => d.classList.remove("active"));
      e.target.classList.add("active");
      this.speedMenu.classList.remove("open");
    });
    document.addEventListener("click", () => this.speedMenu.classList.remove("open"));

    // pip + fullscreen
    this.pipBtn.addEventListener("click", () => this.togglePip());
    this.fsBtn.addEventListener("click", () => this.toggleFullscreen());
    document.addEventListener("fullscreenchange", () => {
      const on = document.fullscreenElement === this.container;
      this.fsBtn.innerHTML = on ? SVG.fsExit : SVG.fsEnter;
      this.container.classList.toggle("fullscreen", on);
    });

    // auto-hide controls
    let hideTimer;
    const show = () => {
      this.container.classList.add("active");
      clearTimeout(hideTimer);
      hideTimer = setTimeout(() => {
        if (!v.paused) this.container.classList.remove("active");
      }, 2500);
    };
    this.container.addEventListener("mousemove", show);
    this.container.addEventListener("mouseleave", () => {
      if (!v.paused) this.container.classList.remove("active");
    });

    // keyboard
    this._onKey = (e) => {
      if (!this.container.isConnected) return;
      if (["INPUT", "TEXTAREA"].includes(document.activeElement?.tagName)) return;
      const k = e.key.toLowerCase();
      if (k === " " || k === "k") { e.preventDefault(); this.toggle(); }
      else if (k === "m") this.toggleMute();
      else if (k === "f") this.toggleFullscreen();
      else if (k === "i") this.togglePip();
      else if (k === "arrowright") v.currentTime = Math.min(v.duration, v.currentTime + 5);
      else if (k === "arrowleft") v.currentTime = Math.max(0, v.currentTime - 5);
      else if (k === "arrowup") { e.preventDefault(); v.volume = Math.min(1, v.volume + 0.05); }
      else if (k === "arrowdown") { e.preventDefault(); v.volume = Math.max(0, v.volume - 0.05); }
      else if (k === "j") v.currentTime = Math.max(0, v.currentTime - 10);
      else if (k === "l") v.currentTime = Math.min(v.duration, v.currentTime + 10);
      else if (k >= "0" && k <= "9") v.currentTime = (parseInt(k, 10) / 10) * v.duration;
    };
    document.addEventListener("keydown", this._onKey);
  }

  _updateVolIcon() {
    this.muteBtn.innerHTML = this.video.muted || this.video.volume === 0 ? SVG.volMute : SVG.volHigh;
  }

  load({ src, poster, title } = {}) {
    if (src) this.video.src = src;
    if (poster !== undefined) this.video.poster = poster || "";
    this.titleEl.textContent = title || "";
    this.titleEl.style.display = title ? "block" : "none";
    if (this.opts.autoplay) this.video.play().catch(() => {});
  }

  play() { return this.video.play(); }
  pause() { this.video.pause(); }
  toggle() { this.video.paused ? this.play() : this.pause(); }
  toggleMute() {
    if (this.video.muted || this.video.volume === 0) {
      this.video.muted = false;
      this.video.volume = this.lastVolume || 1;
    } else {
      this.lastVolume = this.video.volume;
      this.video.muted = true;
    }
  }
  async toggleFullscreen() {
    if (document.fullscreenElement) await document.exitFullscreen();
    else await this.container.requestFullscreen();
  }
  async togglePip() {
    try {
      if (document.pictureInPictureElement) await document.exitPictureInPicture();
      else await this.video.requestPictureInPicture();
    } catch {}
  }

  destroy() {
    document.removeEventListener("keydown", this._onKey);
    this._cleanup?.();
    this.container.innerHTML = "";
    this.container.classList.remove("thp", "playing", "paused", "active", "fullscreen", "loading");
  }
}
