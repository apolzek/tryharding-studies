FROM golang:1.22-alpine AS build
WORKDIR /src
COPY shared/ ./shared/
COPY collector-proxy/ ./collector-proxy/
WORKDIR /src/collector-proxy
RUN go mod download && CGO_ENABLED=0 go build -o /out/collector-proxy .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/collector-proxy /collector-proxy
USER nonroot:nonroot
EXPOSE 8443
ENTRYPOINT ["/collector-proxy"]
