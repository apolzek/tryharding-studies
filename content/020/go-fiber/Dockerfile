FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod .
RUN go mod download || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o loan-gateway .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/loan-gateway .
EXPOSE 8080
CMD ["./loan-gateway"]
