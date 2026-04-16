## giantswarm/mcp-prometheus — with alert notifications

Go server. 18 read-only tools with first-class alerting support. This stack wires in Alertmanager and a rule file that always fires (`AlwaysFiring`), so `get_alerts` / `get_rules` / `get_alertmanagers` return real data.

### Build (image is not published to a public registry)

```sh
git clone --depth=1 https://github.com/giantswarm/mcp-prometheus /tmp/mcp-prometheus
cd /tmp/mcp-prometheus
cat > Dockerfile.local <<'EOF'
FROM golang:1.26.2-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o /out/mcp-prometheus .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/mcp-prometheus /usr/local/bin/mcp-prometheus
ENTRYPOINT ["/usr/local/bin/mcp-prometheus"]
CMD ["serve"]
EOF
docker build -f Dockerfile.local -t local/mcp-prometheus:latest .
```

### Run

```sh
docker compose up -d
# Prometheus:    http://localhost:19093
# Alertmanager:  http://localhost:19094
# MCP (http):    http://localhost:18083/mcp
```

Give Prometheus 10–15 s for the rule to load, then verify the alert is firing:

```sh
curl -s http://localhost:19093/api/v1/alerts | jq '.data.alerts[] | {name:.labels.alertname,state}'
# {"name":"AlwaysFiring","state":"firing"}
```

### Test over stdio

From `content/033/`:

```sh
python3 mcp_client.py \
  --call execute_query '{"query":"up"}' \
  --call get_alerts '{}' \
  --call get_rules '{}' \
  --call get_alertmanagers '{}' \
  --call list_label_names '{}' \
  -- docker run --rm -i --network mcp033-giantswarm_obs \
       -e PROMETHEUS_URL=http://prometheus:9090 \
       local/mcp-prometheus:latest serve --transport=stdio
```

Result: `initOk=true`, 18 tools, all 5 calls succeed. See `../results/giantswarm.json`. `get_alerts` returns `AlwaysFiring` with `severity=critical,team=observability`; `get_alertmanagers` returns the active endpoint `http://alertmanager:9093/api/v2/alerts`.

### Teardown

```sh
docker compose down -v
```
