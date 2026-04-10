import os
import tempfile
from fastapi import FastAPI, UploadFile, File, HTTPException
from faster_whisper import WhisperModel

MODEL_SIZE = os.getenv("WHISPER_MODEL", "large-v3")
DEVICE     = os.getenv("WHISPER_DEVICE", "cuda")
COMPUTE    = os.getenv("WHISPER_COMPUTE", "float16")

print(f"[whisper] Carregando modelo {MODEL_SIZE} em {DEVICE} ({COMPUTE})...")
model = WhisperModel(MODEL_SIZE, device=DEVICE, compute_type=COMPUTE)
print("[whisper] Modelo pronto.")

app = FastAPI()

@app.get("/health")
def health():
    return {"status": "ok", "model": MODEL_SIZE, "device": DEVICE}

@app.post("/transcribe")
async def transcribe(file: UploadFile = File(...), language: str = "pt"):
    suffix = os.path.splitext(file.filename or "audio")[1] or ".ogg"
    with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
        tmp.write(await file.read())
        tmp_path = tmp.name

    try:
        segments, info = model.transcribe(tmp_path, language=language or None)
        text = " ".join(s.text.strip() for s in segments)
        return {
            "text": text,
            "language": info.language,
            "language_probability": round(info.language_probability, 3),
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        os.unlink(tmp_path)
