import argparse
import os
import pickle
import sys
import time
from pathlib import Path

import cv2
import numpy as np
from insightface.app import FaceAnalysis

KNOWN_DIR = Path(os.environ.get("KNOWN_DIR", "/data/known_faces"))
OUTPUT_DIR = Path(os.environ.get("OUTPUT_DIR", "/data/output"))
EMBEDDINGS_FILE = OUTPUT_DIR / "embeddings.pkl"
MATCH_THRESHOLD = float(os.environ.get("MATCH_THRESHOLD", "0.45"))
DET_SIZE = int(os.environ.get("DET_SIZE", "640"))
MODEL_NAME = os.environ.get("MODEL_NAME", "buffalo_l")


def build_app():
    providers = ["CUDAExecutionProvider", "CPUExecutionProvider"]
    app = FaceAnalysis(name=MODEL_NAME, providers=providers)
    app.prepare(ctx_id=0, det_size=(DET_SIZE, DET_SIZE))
    active = [p for p in app.models["detection"].session.get_providers()]
    print(f"[init] model={MODEL_NAME} providers={active}")
    return app


def cosine(a, b):
    return float(np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b)))


def enroll(app):
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    if not KNOWN_DIR.exists():
        print(f"[enroll] missing {KNOWN_DIR}")
        sys.exit(1)

    db = {}
    images = sorted([p for p in KNOWN_DIR.iterdir() if p.suffix.lower() in {".jpg", ".jpeg", ".png"}])
    if not images:
        print(f"[enroll] no images in {KNOWN_DIR}")
        sys.exit(1)

    for img_path in images:
        name = img_path.stem
        img = cv2.imread(str(img_path))
        if img is None:
            print(f"[enroll] skip unreadable: {img_path.name}")
            continue
        faces = app.get(img)
        if not faces:
            print(f"[enroll] no face: {img_path.name}")
            continue
        face = max(faces, key=lambda f: (f.bbox[2] - f.bbox[0]) * (f.bbox[3] - f.bbox[1]))
        db.setdefault(name, []).append(face.normed_embedding)
        print(f"[enroll] {name} <- {img_path.name}")

    if not db:
        print("[enroll] no embeddings produced")
        sys.exit(1)

    db = {n: np.mean(np.stack(v), axis=0) for n, v in db.items()}
    with EMBEDDINGS_FILE.open("wb") as f:
        pickle.dump(db, f)
    print(f"[enroll] wrote {len(db)} identities to {EMBEDDINGS_FILE}")


def load_db():
    if not EMBEDDINGS_FILE.exists():
        print(f"[error] embeddings not found at {EMBEDDINGS_FILE}; run 'enroll' first")
        sys.exit(1)
    with EMBEDDINGS_FILE.open("rb") as f:
        return pickle.load(f)


def identify(face, db):
    best_name, best_score = "unknown", -1.0
    for name, emb in db.items():
        s = cosine(face.normed_embedding, emb)
        if s > best_score:
            best_name, best_score = name, s
    if best_score < MATCH_THRESHOLD:
        return "unknown", best_score
    return best_name, best_score


def annotate(frame, faces, db):
    for face in faces:
        x1, y1, x2, y2 = face.bbox.astype(int)
        name, score = identify(face, db)
        color = (0, 255, 0) if name != "unknown" else (0, 0, 255)
        cv2.rectangle(frame, (x1, y1), (x2, y2), color, 2)
        label = f"{name} {score:.2f}"
        cv2.putText(frame, label, (x1, max(0, y1 - 8)), cv2.FONT_HERSHEY_SIMPLEX, 0.7, color, 2)
    return frame


def open_camera(device):
    cap = cv2.VideoCapture(device, cv2.CAP_V4L2)
    cap.set(cv2.CAP_PROP_FRAME_WIDTH, 1280)
    cap.set(cv2.CAP_PROP_FRAME_HEIGHT, 720)
    if not cap.isOpened():
        print(f"[error] cannot open camera {device}")
        sys.exit(1)
    return cap


def snapshot(app, device):
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    db = load_db()
    cap = open_camera(device)
    for _ in range(5):  # let auto-exposure settle
        cap.read()
    ok, frame = cap.read()
    cap.release()
    if not ok:
        print("[snapshot] failed to capture")
        sys.exit(1)
    faces = app.get(frame)
    print(f"[snapshot] detected {len(faces)} face(s)")
    for f in faces:
        name, score = identify(f, db)
        print(f"  - {name} ({score:.3f})")
    annotated = annotate(frame, faces, db)
    out = OUTPUT_DIR / f"snapshot_{int(time.time())}.jpg"
    cv2.imwrite(str(out), annotated)
    print(f"[snapshot] wrote {out}")


def live(app, device):
    db = load_db()
    cap = open_camera(device)
    print("[live] press q in the window to quit")
    fps_t, frames = time.time(), 0
    while True:
        ok, frame = cap.read()
        if not ok:
            break
        faces = app.get(frame)
        annotated = annotate(frame, faces, db)
        frames += 1
        if frames % 30 == 0:
            now = time.time()
            print(f"[live] {30 / (now - fps_t):.1f} fps")
            fps_t = now
        cv2.imshow("face-recognizer", annotated)
        if cv2.waitKey(1) & 0xFF == ord("q"):
            break
    cap.release()
    cv2.destroyAllWindows()


def main():
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest="cmd", required=True)
    sub.add_parser("enroll")
    p_snap = sub.add_parser("snapshot")
    p_snap.add_argument("--device", default="/dev/video0")
    p_live = sub.add_parser("live")
    p_live.add_argument("--device", default="/dev/video0")
    p_web = sub.add_parser("web")
    p_web.add_argument("--device", default="/dev/video0")
    p_web.add_argument("--host", default="0.0.0.0")
    p_web.add_argument("--port", type=int, default=8000)
    args = parser.parse_args()

    app = build_app()
    if args.cmd == "enroll":
        enroll(app)
    elif args.cmd == "snapshot":
        snapshot(app, args.device)
    elif args.cmd == "live":
        live(app, args.device)
    elif args.cmd == "web":
        import server
        server.start(app, args.device, args.host, args.port)


if __name__ == "__main__":
    main()
