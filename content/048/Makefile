SHELL := /bin/bash
KIND_CLUSTER := obs-saas
KUBECONFIG_CTX := kind-$(KIND_CLUSTER)
COMPOSE := docker compose

.PHONY: help
help:
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: up
up: kind-up control-plane-up frontend-up ## full local stack

.PHONY: kind-up
kind-up: ## create kind cluster + ingress
	@kind get clusters | grep -q '^$(KIND_CLUSTER)$$' || kind create cluster --name $(KIND_CLUSTER) --config deploy/kind/kind-config.yaml
	@kubectl --context $(KUBECONFIG_CTX) apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.2/deploy/static/provider/kind/deploy.yaml
	@kubectl --context $(KUBECONFIG_CTX) -n ingress-nginx wait --for=condition=ready pod -l app.kubernetes.io/component=controller --timeout=180s

.PHONY: control-plane-up
control-plane-up: ## build + run control plane via docker-compose
	$(COMPOSE) up -d --build

.PHONY: frontend-up
frontend-up: ## dev server for frontend
	@cd frontend && npm install && npm run dev -- --host 0.0.0.0 &

.PHONY: signup
signup: ## curl test — register a tenant
	@curl -sS -X POST http://localhost:8080/api/v1/register \
	  -H 'content-type: application/json' \
	  -d '{"email":"demo-$$$$@example.com","password":"hunter22hunter22"}' | jq .

.PHONY: test
test: test-go test-front test-integration ## run all tests

.PHONY: test-go
test-go:
	@cd services && go test ./... -count=1

.PHONY: test-front
test-front:
	@cd frontend && npm run test -- --run

.PHONY: test-integration
test-integration:
	@cd tests/integration && go test -count=1 -timeout 10m ./...

.PHONY: destroy
destroy:
	-$(COMPOSE) down -v
	-kind delete cluster --name $(KIND_CLUSTER)
