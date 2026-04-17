.PHONY: help venv install run run-bg stop status test clean ollama-pull

PY ?= python3
VENV ?= .venv
BIN := $(VENV)/bin
PIP := $(BIN)/pip
PYTHON := $(BIN)/python

PIDFILE ?= .redqueen.pid
LOGFILE ?= .redqueen.log

help:
	@echo "RedQueen targets:"
	@echo "  make install      - create venv and install deps"
	@echo "  make run          - launch the server (foreground)"
	@echo "  make run-bg       - launch in background (logs to $(LOGFILE))"
	@echo "  make stop         - stop the running server"
	@echo "  make status       - show server status"
	@echo "  make test         - run pytest"
	@echo "  make ollama-pull  - pull default llama model"
	@echo "  make clean        - remove venv and caches"

venv:
	$(PY) -m venv $(VENV)

install: venv
	$(PIP) install --upgrade pip
	$(PIP) install -r requirements.txt

run:
	$(PYTHON) run.py

run-bg:
	@if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
		echo "[redqueen] already running (pid $$(cat $(PIDFILE)))"; exit 1; \
	fi
	@nohup $(PYTHON) run.py >$(LOGFILE) 2>&1 & echo $$! > $(PIDFILE)
	@sleep 0.3
	@echo "[redqueen] started (pid $$(cat $(PIDFILE))) — logs: $(LOGFILE)"

stop:
	@if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
		kill $$(cat $(PIDFILE)) && rm -f $(PIDFILE) && echo "[redqueen] stopped"; \
	else \
		echo "[redqueen] no pidfile — trying pkill..."; \
		pkill -f "uvicorn backend.main" 2>/dev/null || pkill -f "run.py" 2>/dev/null || true; \
		rm -f $(PIDFILE); \
		echo "[redqueen] stop signal sent"; \
	fi

status:
	@if [ -f $(PIDFILE) ] && kill -0 $$(cat $(PIDFILE)) 2>/dev/null; then \
		echo "[redqueen] RUNNING (pid $$(cat $(PIDFILE)))"; \
	else \
		echo "[redqueen] not running"; \
	fi

test:
	$(PYTHON) -m pytest -q

ollama-pull:
	ollama pull llama3.2

clean:
	rm -rf $(VENV) .pytest_cache **/__pycache__ $(PIDFILE) $(LOGFILE)
