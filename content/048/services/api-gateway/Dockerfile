FROM golang:1.22-alpine AS build
WORKDIR /src
COPY shared/ ./shared/
COPY api-gateway/ ./api-gateway/
WORKDIR /src/api-gateway
RUN go mod download && CGO_ENABLED=0 go build -o /out/api-gateway .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api-gateway /api-gateway
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api-gateway"]
