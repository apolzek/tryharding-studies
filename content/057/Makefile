# Convenience targets for the 057 playground.
# `make help` shows the full list.

DC = docker compose

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: up
up: ## Build + run everything
	$(DC) up --build -d

.PHONY: down
down: ## Stop everything (keeps volumes)
	$(DC) down

.PHONY: nuke
nuke: ## Stop everything AND drop volumes/networks
	$(DC) down -v --remove-orphans

.PHONY: logs
logs: ## Tail all service logs
	$(DC) logs -f --tail=100

.PHONY: ps
ps: ## List services
	$(DC) ps

.PHONY: seed
seed: ## Populate the system with demo data
	bash scripts/seed.sh

.PHONY: load
load: ## Run the k6 load test against the nginx LB
	$(DC) --profile load run --rm k6 run /scripts/checkout.js

.PHONY: test
test: ## Run all service-level unit tests
	@echo "→ auth"       && cd services/auth       && go test ./... && cd -
	@echo "→ customer"   && cd services/customer   && go test ./... && cd -
	@echo "→ payment"    && cd services/payment    && go test ./... && cd -
	@echo "→ catalog"    && cd services/catalog    && pytest -q && cd -
	@echo "→ signature"  && cd services/signature  && pytest -q && cd -
	@echo "→ notification" && cd services/notification && pytest -q && cd -
	@echo "→ order"      && cd services/order      && pytest -q && cd -
	@echo "→ bff"        && cd services/bff        && npm test && cd -
	@echo "→ cart"       && cd services/cart       && npm test && cd -

.PHONY: chaos-slow
chaos-slow: ## Inject 1.5s latency on order+payment
	curl -s -X POST localhost:8006/chaos/inject -H 'content-type: application/json' -d '{"latency_ms":1500}' | jq .
	curl -s -X POST localhost:8007/chaos/inject -H 'content-type: application/json' -d '{"latency_ms":1500}' | jq .

.PHONY: chaos-errors
chaos-errors: ## Inject 50% error rate on order+payment
	curl -s -X POST localhost:8006/chaos/inject -H 'content-type: application/json' -d '{"error_rate":0.5}' | jq .
	curl -s -X POST localhost:8007/chaos/inject -H 'content-type: application/json' -d '{"error_rate":0.5}' | jq .

.PHONY: chaos-off
chaos-off: ## Disable all chaos
	curl -s -X POST localhost:8006/chaos/inject -H 'content-type: application/json' -d '{"latency_ms":0,"error_rate":0}' | jq .
	curl -s -X POST localhost:8007/chaos/inject -H 'content-type: application/json' -d '{"latency_ms":0,"error_rate":0}' | jq .
