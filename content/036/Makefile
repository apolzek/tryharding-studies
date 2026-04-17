# =========================================================================
# PoC 036 — WebSocket vs Webhook
# =========================================================================
# Targets prefixados com `local-` rodam os servidores direto na sua máquina
# (requerem toolchain instalada). Targets sem prefixo usam docker-compose.
# =========================================================================

SHELL := /bin/bash
SECRET ?= s3cret

# ---------- Docker ----------
.PHONY: build up down logs ps clean
build:            ; docker compose build
up:               ; docker compose up -d
down:             ; docker compose down
ps:               ; docker compose ps
logs:             ; docker compose logs -f --tail=50
clean:            ; docker compose down -v --rmi local

# ---------- Producer ----------
.PHONY: burst ws-notify
burst:            ; docker compose --profile tools run --rm producer webhook --rps 50 --duration 10
ws-notify:        ; docker compose --profile tools run --rm producer ws

# ---------- Stats (webhook counters) ----------
.PHONY: stats
stats:
	@for p in 9001 9002 9003 9004 9005; do \
	  echo "--- :$$p" ; curl -sS http://localhost:$$p/stats || echo "down"; echo; \
	done

# ---------- Local (no docker) ----------
.PHONY: local-ws-go local-ws-py local-ws-js local-wh-go local-wh-py local-wh-js
local-ws-go:   ; cd websocket/go && go run server.go
local-ws-py:   ; cd websocket/python && pip install -q -r requirements.txt && python server.py
local-ws-js:   ; cd websocket/javascript && npm install --silent && node server.js
local-wh-go:   ; cd webhook/go && go run server.go
local-wh-py:   ; cd webhook/python && pip install -q -r requirements.txt && python server.py
local-wh-js:   ; cd webhook/javascript && node server.js

# ---------- Ad-hoc tests ----------
.PHONY: curl-webhook wscat-go
curl-webhook:
	@BODY='{"event":"order.created","id":42}'; \
	SIG=$$(printf '%s' "$$BODY" | openssl dgst -sha256 -hmac $(SECRET) | awk '{print $$2}'); \
	for p in 9001 9002 9003 9004 9005; do \
	  echo "--- POST :$$p"; \
	  curl -sS -X POST http://localhost:$$p/webhook \
	    -H "Content-Type: application/json" -H "X-Signature: $$SIG" -d "$$BODY"; echo; \
	done

# Requires: npm i -g wscat
wscat-go:         ; wscat -c ws://localhost:8001/ws
