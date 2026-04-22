#!/usr/bin/env bash
set -euo pipefail

BASE="http://127.0.0.1:19003"

echo "==> wait for transfer.sh"
for i in $(seq 1 20); do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/" || echo 000)
  if [ "$code" = "200" ] || [ "$code" = "404" ]; then
    echo "ready (HTTP $code)"; break
  fi
  sleep 1
done

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
head -c 4096 /dev/urandom > "$TMP/payload.bin"
SHA_IN=$(sha256sum "$TMP/payload.bin" | awk '{print $1}')
echo "input sha256: $SHA_IN"

echo "==> upload"
URL=$(curl -fsS --upload-file "$TMP/payload.bin" "$BASE/payload.bin")
echo "download url: $URL"

echo "==> download & compare"
curl -fsS -o "$TMP/out.bin" "$URL"
SHA_OUT=$(sha256sum "$TMP/out.bin" | awk '{print $1}')
echo "output sha256: $SHA_OUT"

if [ "$SHA_IN" = "$SHA_OUT" ]; then
  echo "OK: round-trip checksum matches"
else
  echo "FAIL: checksum mismatch"; exit 1
fi
