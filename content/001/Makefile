# Makefile

CADDY_LOAD_BALANCER = ./type-load-balancer/docker-compose-caddy.yaml
CADDY_REVERSE_PROXY = ./type-reverse-proxy/docker-compose-caddy.yaml

ENVOY_LOAD_BALANCER = ./type-load-balancer/docker-compose-envoy.yaml
ENVOY_REVERSE_PROXY = ./type-reverse-proxy/docker-compose-envoy.yaml

HAPROXY_LOAD_BALANCER = ./type-load-balancer/docker-compose-haproxy.yaml
HAPROXY_REVERSE_PROXY = ./type-reverse-proxy/docker-compose-haproxy.yaml

NGINX_LOAD_BALANCER = ./type-load-balancer/docker-compose-nginx.yaml
NGINX_REVERSE_PROXY = ./type-reverse-proxy/docker-compose-nginx.yaml

TRAEFIK_LOAD_BALANCER = ./type-load-balancer/docker-compose-traefik.yaml
TRAEFIK_REVERSE_PROXY = ./type-reverse-proxy/docker-compose-traefik.yaml

.PHONY: up down

up:
ifeq ($(service),caddy_load_balancer)
	docker compose -f $(CADDY_LOAD_BALANCER) up -d
else
ifeq ($(service),caddy_reverse_proxy)
	docker compose -f $(CADDY_REVERSE_PROXY) up -d
else
ifeq ($(service),envoy_load_balancer)
	docker compose -f $(ENVOY_LOAD_BALANCER) up -d
else
ifeq ($(service),envoy_reverse_proxy)
	docker compose -f $(ENVOY_REVERSE_PROXY) up -d
else
ifeq ($(service),haproxy_load_balancer)
	docker compose -f $(HAPROXY_LOAD_BALANCER) up -d
else
ifeq ($(service),haproxy_reverse_proxy)
	docker compose -f $(HAPROXY_REVERSE_PROXY) up -d
else
ifeq ($(service),nginx_load_balancer)
	docker compose -f $(NGINX_LOAD_BALANCER) up -d
else
ifeq ($(service),nginx_reverse_proxy)
	docker compose -f $(NGINX_REVERSE_PROXY) up -d
else
ifeq ($(service),traefik_load_balancer)
	docker compose -f $(TRAEFIK_LOAD_BALANCER) up -d
else
ifeq ($(service),traefik_reverse_proxy)
	docker compose -f $(TRAEFIK_REVERSE_PROXY) up -d
else
	@echo "Invalid Option. Ex: caddy_load_balancer, caddy_reverse_proxy"
endif
endif
endif
endif
endif
endif
endif
endif
endif
endif

down:
ifeq ($(service),caddy_load_balancer)
	docker compose -f $(CADDY_LOAD_BALANCER) down
else
ifeq ($(service),caddy_reverse_proxy)
	docker compose -f $(CADDY_REVERSE_PROXY) down
else
ifeq ($(service),envoy_load_balancer)
	docker compose -f $(ENVOY_LOAD_BALANCER) down
else
ifeq ($(service),envoy_reverse_proxy)
	docker compose -f $(ENVOY_REVERSE_PROXY) down
else
ifeq ($(service),haproxy_load_balancer)
	docker compose -f $(HAPROXY_LOAD_BALANCER) down
else
ifeq ($(service),haproxy_reverse_proxy)
	docker compose -f $(HAPROXY_REVERSE_PROXY) down
else
ifeq ($(service),nginx_load_balancer)
	docker compose -f $(NGINX_LOAD_BALANCER) down
else
ifeq ($(service),nginx_reverse_proxy)
	docker compose -f $(NGINX_REVERSE_PROXY) down
else
ifeq ($(service),traefik_load_balancer)
	docker compose -f $(TRAEFIK_LOAD_BALANCER) down
else
ifeq ($(service),traefik_reverse_proxy)
	docker compose -f $(TRAEFIK_REVERSE_PROXY) down
else
	@echo "Invalid Option. Ex: caddy_load_balancer, caddy_reverse_proxy"
endif
endif
endif
endif
endif
endif
endif
endif
endif
endif