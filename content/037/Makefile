SHELL := /usr/bin/env bash
COMPOSE ?= docker compose

.PHONY: build up down logs ps kubeconfig kubectl nodes test clean

build:
	$(COMPOSE) build

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f --tail=100

kubeconfig:
	docker exec k0s-controller k0s kubeconfig admin > kubeconfig
	sed -i 's#https://[^:]*:6443#https://127.0.0.1:6443#' kubeconfig
	@echo "KUBECONFIG=$$PWD/kubeconfig"

kubectl:
	docker exec -it k0s-controller k0s kubectl $(filter-out $@,$(MAKECMDGOALS))

nodes:
	docker exec k0s-controller k0s kubectl get nodes -o wide

test:
	bash scripts/test-cluster.sh

clean:
	$(COMPOSE) down -v
	rm -f kubeconfig

%:
	@:
