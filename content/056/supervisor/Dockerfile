# Build opampsupervisor from opentelemetry-collector-contrib source.
ARG CONTRIB_VERSION=v0.117.0

FROM golang:1.24-alpine AS build
ARG CONTRIB_VERSION
RUN apk add --no-cache git
WORKDIR /src
RUN git clone --depth 1 --branch ${CONTRIB_VERSION} \
      https://github.com/open-telemetry/opentelemetry-collector-contrib.git .
WORKDIR /src/cmd/opampsupervisor
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/opampsupervisor .

# Final — based on the contrib collector image so /otelcol-contrib is available.
# Image is distroless-ish (no shell, no /etc/passwd); avoid RUN/USER directives.
# Mount points are auto-created by Docker; run as UID 0 via compose (user: "0:0")
# so the named volume at /var/lib/otelcol/supervisor is writable.
FROM otel/opentelemetry-collector-contrib:0.117.0
COPY --from=build /out/opampsupervisor /usr/local/bin/opampsupervisor
ENTRYPOINT ["/usr/local/bin/opampsupervisor"]
CMD ["--config", "/etc/otelcol-supervisor/supervisor.yaml"]
