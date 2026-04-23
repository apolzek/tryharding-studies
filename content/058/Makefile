SHELL := /bin/bash
APP_HOST ?= 127.0.0.1
APP_PORT ?= 8765
BASE     := http://$(APP_HOST):$(APP_PORT)
DUR      ?= 15s
VUS      ?= 50
RPS      ?= 500
RESULTS  := $(CURDIR)/results

# Use --network host everywhere so tools connect directly to the host's 127.0.0.1:8765.
# That avoids docker bridge NAT and port conflicts between parallel test runs.
NET        := --network host
RM         := --rm
MOUNT_LT   := -v $(CURDIR)/loadtests:/lt:ro
MOUNT_OUT  := -v $(RESULTS):/out
DOCKER     := docker run $(RM) $(NET) $(MOUNT_LT) $(MOUNT_OUT)

.PHONY: help build up down restart logs wait-healthy smoke \
        test-all test-k6 test-vegeta test-wrk test-hey test-ab \
        test-bombardier test-siege test-artillery test-locust test-drill \
        clean-results report

help:
	@echo "Targets:"
	@echo "  build           build the Go app image"
	@echo "  up / down       start/stop bigbench (host network)"
	@echo "  smoke           curl the three endpoints"
	@echo "  test-all        run all 10 load tools sequentially"
	@echo "  test-<tool>     run a single tool (k6|vegeta|wrk|hey|ab|bombardier|siege|artillery|locust|drill)"
	@echo "  report          regenerate results summary in README.md"

build:
	docker compose build

up: build
	docker compose up -d
	$(MAKE) wait-healthy

down:
	docker compose down -v

restart: down up

logs:
	docker compose logs -f --tail=100 bigbench

wait-healthy:
	@echo "waiting for $(BASE)/health ..."
	@for i in $$(seq 1 30); do \
	  curl -fsS $(BASE)/health >/dev/null 2>&1 && echo "ready" && exit 0; \
	  sleep 1; \
	done; \
	echo "bigbench failed to become ready" && exit 1

smoke:
	@curl -s $(BASE)/simple     | head -c 200 && echo
	@curl -s $(BASE)/medium?n=2000 | head -c 200 && echo
	@curl -s $(BASE)/heavy?n=25 | head -c 200 && echo

clean-results:
	rm -rf $(RESULTS) && mkdir -p $(RESULTS)

# --------- 1. k6 ---------
test-k6:
	mkdir -p $(RESULTS)
	$(DOCKER) -e BASE=$(BASE) grafana/k6:0.50.0 run --summary-export=/out/k6.json /lt/k6/script.js \
	  2>&1 | tee $(RESULTS)/k6.txt

# --------- 2. Vegeta ---------
test-vegeta:
	mkdir -p $(RESULTS)
	$(DOCKER) --entrypoint sh peterevans/vegeta:latest -c '\
	  for ep in simple medium heavy; do \
	    case $$ep in \
	      simple) url=$(BASE)/simple ;; \
	      medium) url=$(BASE)/medium?n=2000 ;; \
	      heavy)  url=$(BASE)/heavy?n=25 ;; \
	    esac; \
	    echo GET $$url | vegeta attack -rate=$(RPS) -duration=$(DUR) | \
	      tee /out/vegeta-$$ep.bin | vegeta report | tee /out/vegeta-$$ep.txt; \
	  done'

# --------- 3. wrk ---------
test-wrk:
	mkdir -p $(RESULTS)
	@for ep in simple medium heavy; do \
	  case $$ep in \
	    simple) path=/simple ;; \
	    medium) path="/medium?n=2000" ;; \
	    heavy)  path="/heavy?n=25" ;; \
	  esac; \
	  echo "=== wrk $$ep ==="; \
	  $(DOCKER) williamyeh/wrk:latest -t4 -c$(VUS) -d$(DUR) --latency "$(BASE)$$path" \
	    2>&1 | tee $(RESULTS)/wrk-$$ep.txt; \
	done

# --------- 4. hey ---------
# rcmorano/docker-hey and williamyeh/hey both exist; use rakyll's mirror via alpine install fallback
test-hey:
	mkdir -p $(RESULTS)
	@for ep in simple medium heavy; do \
	  case $$ep in \
	    simple) path=/simple ;; \
	    medium) path="/medium?n=2000" ;; \
	    heavy)  path="/heavy?n=25" ;; \
	  esac; \
	  echo "=== hey $$ep ==="; \
	  $(DOCKER) rcmorano/docker-hey -z $(DUR) -c $(VUS) "$(BASE)$$path" \
	    2>&1 | tee $(RESULTS)/hey-$$ep.txt; \
	done

# --------- 5. Apache Bench (ab) ---------
test-ab:
	mkdir -p $(RESULTS)
	@for ep in simple medium heavy; do \
	  case $$ep in \
	    simple) path=/simple ;; \
	    medium) path="/medium?n=2000" ;; \
	    heavy)  path="/heavy?n=25" ;; \
	  esac; \
	  echo "=== ab $$ep ==="; \
	  $(DOCKER) --entrypoint ab httpd:2.4-alpine -n 20000 -c $(VUS) -t 15 "$(BASE)$$path" \
	    2>&1 | tee $(RESULTS)/ab-$$ep.txt; \
	done

# --------- 6. Bombardier ---------
test-bombardier:
	mkdir -p $(RESULTS)
	@for ep in simple medium heavy; do \
	  case $$ep in \
	    simple) path=/simple ;; \
	    medium) path="/medium?n=2000" ;; \
	    heavy)  path="/heavy?n=25" ;; \
	  esac; \
	  echo "=== bombardier $$ep ==="; \
	  $(DOCKER) alpine/bombardier:latest -c $(VUS) -d $(DUR) -l "$(BASE)$$path" \
	    2>&1 | tee $(RESULTS)/bombardier-$$ep.txt; \
	done

# --------- 7. Siege ---------
test-siege:
	mkdir -p $(RESULTS)
	$(DOCKER) jstarcher/siege:latest -b -t15s -c $(VUS) -f /lt/siege/urls.txt \
	  2>&1 | tee $(RESULTS)/siege.txt

# --------- 8. Artillery ---------
test-artillery:
	mkdir -p $(RESULTS)
	$(DOCKER) artilleryio/artillery:latest run --output /out/artillery.json /lt/artillery/config.yml \
	  2>&1 | tee $(RESULTS)/artillery.txt

# --------- 9. Locust (headless) ---------
test-locust:
	mkdir -p $(RESULTS)
	$(DOCKER) locustio/locust:2.24.0 -f /lt/locust/locustfile.py \
	  --headless -u $(VUS) -r 25 -t $(DUR) --host=$(BASE) --only-summary \
	  --csv=/out/locust 2>&1 | tee $(RESULTS)/locust.txt

# --------- 10. Drill ---------
test-drill:
	mkdir -p $(RESULTS)
	$(DOCKER) -w /lt/drill xridge/drill:latest --benchmark benchmark.yml --stats \
	  2>&1 | tee $(RESULTS)/drill.txt

test-all: clean-results wait-healthy
	@echo "### Running all 10 tools — approx 3-4 minutes"
	-$(MAKE) test-k6
	-$(MAKE) test-vegeta
	-$(MAKE) test-wrk
	-$(MAKE) test-hey
	-$(MAKE) test-ab
	-$(MAKE) test-bombardier
	-$(MAKE) test-siege
	-$(MAKE) test-artillery
	-$(MAKE) test-locust
	-$(MAKE) test-drill
	@$(MAKE) report

report:
	@echo "raw outputs in $(RESULTS)/ — see README.md for the summary table"
