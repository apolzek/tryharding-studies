FROM golang:1.25.0-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o genmetrics .

FROM alpine:latest

RUN addgroup -g 1001 appgroup && \
    adduser -D -u 1001 -G appgroup appuser

COPY --from=builder /app/genmetrics /usr/local/bin/genmetrics
COPY metrics.yaml /app/metrics.yaml

WORKDIR /app
RUN chown -R appuser:appgroup /app

USER appuser

ENV PORT=8090
EXPOSE 8090

CMD ["genmetrics"]
