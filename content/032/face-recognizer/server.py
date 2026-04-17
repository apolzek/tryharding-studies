import threading
import time
from pathlib import Path

import cv2
import uvicorn
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from fastapi.responses import HTMLResponse, StreamingResponse

import main as core
import objects as obj

api = FastAPI()

_state = {
    "face_app": None,
    "obj_detector": None,
    "db": {},
    "latest_jpeg": None,
}
_db_lock = threading.Lock()
_frame_lock = threading.Lock()


def _camera_loop(device: str):
    cap = core.open_camera(device)
    while True:
        ok, frame = cap.read()
        if not ok:
            time.sleep(0.05)
            continue
        faces = _state["face_app"].get(frame)
        with _db_lock:
            db_snapshot = dict(_state["db"])
        annotated = core.annotate(frame, faces, db_snapshot) if db_snapshot else frame
        if _state["obj_detector"] is not None:
            detections = _state["obj_detector"].detect(frame)
            annotated = obj.draw(annotated, detections)
        ok, buf = cv2.imencode(".jpg", annotated, [cv2.IMWRITE_JPEG_QUALITY, 80])
        if ok:
            with _frame_lock:
                _state["latest_jpeg"] = buf.tobytes()


def _mjpeg():
    boundary = b"--frame"
    while True:
        with _frame_lock:
            data = _state["latest_jpeg"]
        if data is None:
            time.sleep(0.05)
            continue
        yield boundary + b"\r\nContent-Type: image/jpeg\r\n\r\n" + data + b"\r\n"
        time.sleep(0.03)


INDEX = """<!doctype html>
<html><head><title>Face Recognizer</title>
<style>
body{font-family:system-ui,sans-serif;max-width:960px;margin:2em auto;padding:0 1em;color:#222}
img{max-width:100%;border:1px solid #ccc;border-radius:4px}
form{display:flex;gap:.5em;align-items:center;flex-wrap:wrap;margin:1em 0}
input,button{padding:.5em;font-size:1em}
button{cursor:pointer;background:#2563eb;color:#fff;border:0;border-radius:4px}
.tag{display:inline-block;padding:.3em .7em;background:#eef;border-radius:4px;margin:.2em}
.tag button{background:#ef4444;margin-left:.5em;padding:.1em .4em;font-size:.8em}
#status{min-height:1.5em;font-family:monospace;color:#555}
</style></head><body>
<h1>Face Recognizer</h1>
<img src="/stream" alt="live stream" />
<form id="f">
  <input name="name" placeholder="identity name" required pattern="[A-Za-z0-9_\\-]+" />
  <input type="file" name="file" accept="image/jpeg,image/png" required />
  <button type="submit">Upload &amp; re-enroll</button>
</form>
<div id="status"></div>
<h3>Known identities</h3>
<div id="ids"></div>
<script>
async function refresh(){
  const r=await fetch('/identities'); const j=await r.json();
  document.getElementById('ids').innerHTML = j.identities.length
    ? j.identities.map(n=>`<span class="tag">${n}<button data-n="${n}">x</button></span>`).join('')
    : '<em>none yet</em>';
  document.querySelectorAll('.tag button').forEach(b=>b.onclick=async()=>{
    if(!confirm('Delete '+b.dataset.n+'?')) return;
    await fetch('/identities/'+encodeURIComponent(b.dataset.n),{method:'DELETE'});
    refresh();
  });
}
document.getElementById('f').onsubmit=async e=>{
  e.preventDefault();
  const s=document.getElementById('status'); s.textContent='uploading...';
  const r=await fetch('/upload',{method:'POST',body:new FormData(e.target)});
  const j=await r.json(); s.textContent=JSON.stringify(j);
  refresh();
};
refresh();
</script></body></html>"""


@api.get("/", response_class=HTMLResponse)
def index():
    return INDEX


@api.get("/stream")
def stream():
    return StreamingResponse(_mjpeg(), media_type="multipart/x-mixed-replace; boundary=frame")


@api.get("/identities")
def identities():
    with _db_lock:
        return {"identities": sorted(_state["db"].keys())}


@api.post("/upload")
async def upload(name: str = Form(...), file: UploadFile = File(...)):
    safe = "".join(c for c in name if c.isalnum() or c in "-_").strip()
    if not safe:
        raise HTTPException(400, "invalid name")
    ext = Path(file.filename or "").suffix.lower()
    if ext not in {".jpg", ".jpeg", ".png"}:
        raise HTTPException(400, "only jpg/jpeg/png")
    for old in core.KNOWN_DIR.glob(f"{safe}.*"):
        old.unlink()
    dest = core.KNOWN_DIR / f"{safe}{ext}"
    dest.write_bytes(await file.read())
    core.enroll(_state["face_app"])
    with _db_lock:
        _state["db"] = core.load_db()
    return {"ok": True, "name": safe, "identities": sorted(_state["db"].keys())}


@api.delete("/identities/{name}")
def delete_identity(name: str):
    safe = "".join(c for c in name if c.isalnum() or c in "-_").strip()
    removed = [p.name for p in core.KNOWN_DIR.glob(f"{safe}.*")]
    for p in core.KNOWN_DIR.glob(f"{safe}.*"):
        p.unlink()
    if not removed:
        raise HTTPException(404, "no such identity")
    remaining = list(core.KNOWN_DIR.glob("*.jpg")) + list(core.KNOWN_DIR.glob("*.jpeg")) + list(core.KNOWN_DIR.glob("*.png"))
    with _db_lock:
        if remaining:
            core.enroll(_state["face_app"])
            _state["db"] = core.load_db()
        else:
            _state["db"] = {}
            if core.EMBEDDINGS_FILE.exists():
                core.EMBEDDINGS_FILE.unlink()
    return {"ok": True, "removed": removed}


def start(face_app, device: str, host: str = "0.0.0.0", port: int = 8000):
    _state["face_app"] = face_app
    if core.EMBEDDINGS_FILE.exists():
        _state["db"] = core.load_db()
    import os
    if os.environ.get("YOLO_ENABLED", "1") != "0":
        _state["obj_detector"] = obj.ObjectDetector()
    threading.Thread(target=_camera_loop, args=(device,), daemon=True).start()
    print(f"[web] serving on http://{host}:{port}")
    uvicorn.run(api, host=host, port=port, log_level="info")
