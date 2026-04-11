#!/usr/bin/env python3
"""
Testa a API de transcrição local.
Uso:
  python test.py                      # usa arquivo de exemplo gerado automaticamente
  python test.py caminho/para/audio   # usa arquivo existente
"""
import sys
import os
import tempfile
import urllib.request

BASE_URL = "http://localhost:8000"

def check_health():
    with urllib.request.urlopen(f"{BASE_URL}/health") as r:
        import json
        data = json.loads(r.read())
        print(f"[health] {data}")

def transcribe(filepath):
    import urllib.request, json
    boundary = "----FormBoundary7MA4YWxkTrZu0gW"
    filename = os.path.basename(filepath)

    with open(filepath, "rb") as f:
        file_data = f.read()

    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="file"; filename="{filename}"\r\n'
        f"Content-Type: application/octet-stream\r\n\r\n"
    ).encode() + file_data + f"\r\n--{boundary}--\r\n".encode()

    req = urllib.request.Request(
        f"{BASE_URL}/transcribe",
        data=body,
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
        method="POST",
    )
    with urllib.request.urlopen(req) as r:
        return json.loads(r.read())

def generate_test_audio():
    """Gera um arquivo WAV simples com silêncio para testar o endpoint (sem dependências externas)."""
    import struct, wave
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        path = tmp.name
    with wave.open(path, "w") as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(16000)
        wf.writeframes(struct.pack("<" + "h" * 16000, *([0] * 16000)))  # 1s de silêncio
    return path

if __name__ == "__main__":
    print("=== Teste da API Whisper local ===\n")

    print("1. Verificando health...")
    try:
        check_health()
    except Exception as e:
        print(f"[ERRO] API não está rodando: {e}")
        print("  Suba o servidor primeiro: ./start.sh")
        sys.exit(1)

    if len(sys.argv) > 1:
        audio_path = sys.argv[1]
        cleanup = False
    else:
        print("\n2. Nenhum arquivo fornecido — gerando WAV de teste (1s silêncio)...")
        audio_path = generate_test_audio()
        cleanup = True
        print(f"   Arquivo: {audio_path}")
        print("\n   Dica: passe um arquivo de voz real para testar transcrição:")
        print("   python test.py ~/meu_audio.ogg\n")

    print(f"3. Enviando para /transcribe: {os.path.basename(audio_path)}")
    try:
        result = transcribe(audio_path)
        print(f"\n[RESULTADO]")
        print(f"  Texto      : {result['text'] or '(silêncio / sem fala detectada)'}")
        print(f"  Idioma     : {result['language']} (prob: {result['language_probability']})")
    except Exception as e:
        print(f"[ERRO] {e}")
    finally:
        if cleanup:
            os.unlink(audio_path)
