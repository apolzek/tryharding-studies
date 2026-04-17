.PHONY: all build backend agent-deps run-backend run-agent run-agent-dryrun \
        seed-fake test clean install-agent

all: build

# --- build ---
build: backend

backend:
	cd backend && go build -o ../bin/userwatch-backend .

# --- run ---
run-backend:
	./bin/userwatch-backend --addr :8080 --db ./userwatch.db --frontend ./frontend

run-agent: agent-deps
	sudo UW_BACKEND=http://127.0.0.1:8080 UW_TOKEN=dev-token \
	  python3 agent/agent.py --interval 5 -v

run-agent-dryrun: agent-deps
	sudo python3 agent/agent.py --interval 5 --dry-run -v

agent-deps:
	@dpkg -l | grep -q python3-bpfcc || \
	  (echo "Install BCC: sudo apt-get install -y bpfcc-tools python3-bpfcc linux-headers-$$(uname -r)"; exit 1)

# Fake sample generator, for testing the backend + UI without running the agent.
seed-fake:
	@./scripts/seed_fake.sh

test:
	cd backend && go vet ./... && go test ./...

install-agent:
	sudo ./agent/install.sh

clean:
	rm -rf bin userwatch.db
