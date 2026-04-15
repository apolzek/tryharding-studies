SHELL := /usr/bin/env bash
COMPOSE ?= docker compose
PROJECT ?= poc-031

.DEFAULT_GOAL := help

.PHONY: help up down logs ps restart clean test load minio-setup build

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all service images
	$(COMPOSE) build

up: ## Start the full stack in detached mode
	$(COMPOSE) up -d
	@$(MAKE) minio-setup
	@echo ""
	@echo "=== PoC-031 is up ==="
	@echo "Grafana        http://localhost:3000"
	@echo "Prometheus     http://localhost:9090"
	@echo "Tempo          http://localhost:3200"
	@echo "Loki           http://localhost:3100"
	@echo "MinIO Console  http://localhost:9001"
	@echo "Alloy UI       http://localhost:12345"
	@echo "React FE       http://localhost:5173"
	@echo "Vue FE         http://localhost:5174"
	@echo "Vanilla FE     http://localhost:5175"
	@echo "Backend API    http://localhost:8080"

down: ## Stop the stack
	$(COMPOSE) down

logs: ## Tail logs from all services
	$(COMPOSE) logs -f --tail=200

ps: ## Show service status
	$(COMPOSE) ps

restart: ## Restart the stack
	$(COMPOSE) restart

clean: ## Stop and remove volumes (destructive)
	$(COMPOSE) down -v --remove-orphans

test: ## Run Playwright end-to-end tests
	cd tests && npx playwright test

load: ## Run k6 load generator
	cd tests && k6 run load.js

minio-setup: ## Create rrweb-sessions bucket in MinIO
	@echo "Creating MinIO bucket 'rrweb-sessions'..."
	@docker run --rm --network obs-net \
		-e MC_HOST_local=http://minioadmin:minioadmin@minio:9000 \
		minio/mc:latest \
		sh -c "until mc ls local >/dev/null 2>&1; do sleep 1; done; \
		       mc mb -p local/rrweb-sessions || true; \
		       mc anonymous set download local/rrweb-sessions || true" \
		|| echo "minio-setup: bucket creation skipped (already exists or minio not ready)"
