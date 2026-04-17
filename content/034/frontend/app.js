// ================================================================
// RedQueen frontend — WS chat + voice + hacker effects
// ================================================================
(() => {
  const $ = (id) => document.getElementById(id);
  const consoleEl = $("console");
  const promptEl = $("prompt");
  const form = $("prompt-form");
  const micBtn = $("mic");
  const statStatus = $("stat-status").querySelector("b");
  const statModel = $("stat-model").querySelector("b");
  const statLat = $("stat-latency").querySelector("b");
  const statTime = $("stat-time");
  const wsState = $("ws-state").querySelector("b");
  const micState = $("mic-state").querySelector("b");
  const ttsState = $("tts-state").querySelector("b");

  let ws = null;
  let currentLine = null;
  let streaming = false;
  let startTs = 0;
  let cfg = null;
  let history = [];

  // ---------------- CONFIG ----------------
  async function loadConfig() {
    try {
      const r = await fetch("/api/config");
      cfg = await r.json();
      document.title = cfg.ui.title;
      applyTheme(cfg.ui.theme);
      renderAscii(cfg.ui.ascii_art);
      statModel.textContent = cfg.llm_model || "--";
      return cfg;
    } catch (e) {
      sysLine("[config load failed: " + e.message + "]", "error");
      return null;
    }
  }

  function applyTheme(t) {
    const root = document.documentElement.style;
    root.setProperty("--red", t.primary);
    root.setProperty("--red-2", t.secondary);
    root.setProperty("--bg", t.background);
    root.setProperty("--red-dim", t.accent);
    root.setProperty("--text", t.text);
  }

  function renderAscii(key) {
    const art = (window.REDQUEEN_ASCII || {})[key] || window.REDQUEEN_ASCII.red_queen;
    $("ascii-art").textContent = art;
  }

  // ---------------- CONSOLE ----------------
  function line(text, cls = "system") {
    const el = document.createElement("div");
    el.className = "line " + cls;
    el.textContent = text;
    consoleEl.appendChild(el);
    consoleEl.scrollTop = consoleEl.scrollHeight;
    return el;
  }
  const sysLine = (t, c = "system") => line(t, c);

  function typewriter(el, text, speed = 14) {
    return new Promise((res) => {
      el.classList.add("cursor");
      let i = 0;
      const tick = () => {
        el.textContent = text.slice(0, i++);
        consoleEl.scrollTop = consoleEl.scrollHeight;
        if (i <= text.length) setTimeout(tick, speed);
        else { el.classList.remove("cursor"); res(); }
      };
      tick();
    });
  }

  // ---------------- WEBSOCKET ----------------
  function connect() {
    const proto = location.protocol === "https:" ? "wss" : "ws";
    ws = new WebSocket(`${proto}://${location.host}/ws`);
    ws.addEventListener("open", () => {
      wsState.textContent = "online";
      statStatus.textContent = "ONLINE";
    });
    ws.addEventListener("close", () => {
      wsState.textContent = "offline";
      statStatus.textContent = "DISCONNECTED";
      setTimeout(connect, 1500);
    });
    ws.addEventListener("error", () => sysLine("[ws error]", "error"));
    ws.addEventListener("message", (e) => onMsg(JSON.parse(e.data)));
  }

  function onMsg(m) {
    switch (m.type) {
      case "ready":
        statModel.textContent = m.model;
        break;
      case "start":
        streaming = true;
        startTs = performance.now();
        currentLine = document.createElement("div");
        currentLine.className = "line assistant cursor";
        currentLine.textContent = "";
        consoleEl.appendChild(currentLine);
        break;
      case "token":
        if (currentLine) {
          currentLine.textContent += m.token;
          consoleEl.scrollTop = consoleEl.scrollHeight;
        }
        break;
      case "end":
        streaming = false;
        if (currentLine) currentLine.classList.remove("cursor");
        statLat.textContent = Math.round(performance.now() - startTs) + " ms";
        speak(m.reply || (currentLine && currentLine.textContent) || "");
        history.push({ role: "user", content: lastUserPrompt });
        history.push({ role: "assistant", content: m.reply });
        currentLine = null;
        break;
      case "error":
        streaming = false;
        if (currentLine) currentLine.classList.remove("cursor");
        sysLine("error: " + m.error, "error");
        break;
      case "reset_ok":
        history = [];
        sysLine("[memory purged]");
        break;
    }
  }

  let lastUserPrompt = "";
  function send(prompt) {
    if (!ws || ws.readyState !== 1) {
      sysLine("[offline]", "error");
      return;
    }
    lastUserPrompt = prompt;
    line(prompt, "user");
    ws.send(JSON.stringify({ type: "chat", prompt }));
  }

  // ---------------- FORM ----------------
  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const v = promptEl.value.trim();
    if (!v) return;
    promptEl.value = "";
    if (v === "/reset") { ws && ws.send(JSON.stringify({ type: "reset" })); return; }
    if (v === "/stop") { stopSpeak(); return; }
    if (v === "/clear") { consoleEl.innerHTML = ""; return; }
    send(v);
  });

  // ---------------- VOICE (STT) ----------------
  const SR = window.SpeechRecognition || window.webkitSpeechRecognition;
  let recog = null, listening = false;
  function initSTT() {
    if (!SR) { micState.textContent = "unsupported"; micBtn.disabled = true; return; }
    recog = new SR();
    recog.continuous = false;
    recog.interimResults = true;
    recog.lang = (cfg && cfg.voice && cfg.voice.browser.stt_lang) || "en-US";
    recog.onstart = () => { listening = true; micState.textContent = "listening"; micBtn.classList.add("active"); };
    recog.onend = () => { listening = false; micState.textContent = "idle"; micBtn.classList.remove("active"); };
    recog.onerror = (e) => sysLine("[stt: " + e.error + "]", "error");
    recog.onresult = (e) => {
      let finalT = "";
      for (let i = e.resultIndex; i < e.results.length; i++) {
        const t = e.results[i][0].transcript;
        if (e.results[i].isFinal) finalT += t;
        else promptEl.value = t;
      }
      if (finalT) {
        promptEl.value = "";
        send(finalT.trim());
      }
    };
  }
  function toggleMic() {
    if (!recog) return;
    if (listening) recog.stop(); else { try { recog.start(); } catch {} }
  }
  micBtn.addEventListener("click", toggleMic);

  // ---------------- VOICE (TTS) ----------------
  let voices = [];
  function loadVoices() {
    voices = speechSynthesis.getVoices();
  }
  if ("speechSynthesis" in window) {
    loadVoices();
    speechSynthesis.onvoiceschanged = loadVoices;
  }
  function pickVoice() {
    if (!voices.length) return null;
    const wanted = (cfg && cfg.voice && cfg.voice.browser.tts_voice) || "";
    if (wanted) {
      const v = voices.find((x) => x.name === wanted);
      if (v) return v;
    }
    // prefer an English female-sounding voice
    return voices.find((v) => /en.*(female|samantha|zira|victoria)/i.test(v.name))
      || voices.find((v) => v.lang.startsWith("en"))
      || voices[0];
  }
  function speak(text) {
    if (!("speechSynthesis" in window) || !text) return;
    stopSpeak();
    const u = new SpeechSynthesisUtterance(text);
    const v = pickVoice(); if (v) u.voice = v;
    u.rate = (cfg && cfg.voice && cfg.voice.browser.tts_rate) || 1.0;
    u.pitch = (cfg && cfg.voice && cfg.voice.browser.tts_pitch) || 0.6;
    ttsState.textContent = "speaking";
    u.onend = () => (ttsState.textContent = "ready");
    speechSynthesis.speak(u);
  }
  function stopSpeak() {
    if ("speechSynthesis" in window) speechSynthesis.cancel();
    ttsState.textContent = "ready";
  }

  // ---------------- KEYBOARD ----------------
  document.addEventListener("keydown", (e) => {
    if (e.code === "Space" && document.activeElement !== promptEl) {
      e.preventDefault(); toggleMic();
    }
    if (e.key === "Escape") stopSpeak();
  });

  // ---------------- CLOCK ----------------
  setInterval(() => {
    const d = new Date();
    statTime.textContent = d.toTimeString().slice(0, 8);
  }, 1000);

  // ---------------- HUD BARS ----------------
  function jitterBars() {
    const set = (id, v) => ($(id).style.width = Math.max(5, Math.min(99, v)) + "%");
    set("bar-cpu", 40 + Math.random() * 40);
    set("bar-mem", 55 + Math.random() * 30);
    set("bar-net", 10 + Math.random() * 80);
    set("bar-hive", 80 + Math.random() * 18);
  }
  setInterval(jitterBars, 900);
  jitterBars();

  // ---------------- MATRIX RAIN ----------------
  const canvas = $("matrix");
  const ctx = canvas.getContext("2d");
  let chars = "アカサタナハマヤラワ0123456789ABCDEF<>*+-%@#$!?";
  let cols = 0, drops = [];
  function sizeCanvas() {
    canvas.width = innerWidth; canvas.height = innerHeight;
    cols = Math.floor(canvas.width / 14);
    drops = Array(cols).fill(1);
  }
  window.addEventListener("resize", sizeCanvas);
  sizeCanvas();
  function rain() {
    ctx.fillStyle = "rgba(5,0,10,0.08)";
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.fillStyle = "#ff0033";
    ctx.font = "14px monospace";
    ctx.shadowColor = "#ff0033";
    ctx.shadowBlur = 6;
    for (let i = 0; i < cols; i++) {
      const c = chars[(Math.random() * chars.length) | 0];
      ctx.fillText(c, i * 14, drops[i] * 14);
      if (drops[i] * 14 > canvas.height && Math.random() > 0.975) drops[i] = 0;
      drops[i]++;
    }
    requestAnimationFrame(rain);
  }
  rain();

  // ---------------- BOOT SEQUENCE ----------------
  async function bootSequence() {
    const log = $("boot-log");
    const lines = window.REDQUEEN_BOOT || [];
    for (const l of lines) {
      log.textContent += l + "\n";
      await new Promise((r) => setTimeout(r, 90 + Math.random() * 120));
    }
    await new Promise((r) => setTimeout(r, 450));
    $("boot").classList.add("done");
  }

  // ---------------- GREETING ----------------
  async function greet() {
    if (!cfg || !cfg.ui || !cfg.ui.greeting) return;
    const lines = cfg.ui.greeting.split("\n");
    for (const t of lines) {
      if (!t.trim()) { line(" "); continue; }
      const el = document.createElement("div");
      el.className = "line assistant";
      consoleEl.appendChild(el);
      await typewriter(el, t, 18);
    }
    speak(cfg.ui.greeting.replace(/[\[\]]/g, ""));
  }

  // ---------------- INIT ----------------
  (async () => {
    await bootSequence();
    await loadConfig();
    initSTT();
    connect();
    await greet();
  })();
})();
