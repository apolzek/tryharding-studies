# Makefile

COMPOSE = docker compose

.PHONY: build enroll snapshot live web shell down clean xhost

build:
	$(COMPOSE) build

enroll:
	$(COMPOSE) run --rm face-recognizer enroll

snapshot:
	$(COMPOSE) run --rm face-recognizer snapshot

xhost:
	xhost +local:docker

live: xhost
	$(COMPOSE) run --rm face-recognizer live

web:
	$(COMPOSE) run --rm --service-ports face-recognizer web

shell:
	$(COMPOSE) run --rm --entrypoint bash face-recognizer

down:
	$(COMPOSE) down

clean:
	rm -f output/embeddings.pkl output/snapshot_*.jpg
