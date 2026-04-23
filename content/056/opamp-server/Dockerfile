FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download github.com/open-telemetry/opamp-go \
 && go mod download gopkg.in/yaml.v3 || true
COPY . .
RUN go mod tidy \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/opampsrv .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/opampsrv /usr/local/bin/opampsrv
COPY web /web
ENV DATA_DIR=/data
EXPOSE 4320 4321
VOLUME /data
ENTRYPOINT ["/usr/local/bin/opampsrv"]
