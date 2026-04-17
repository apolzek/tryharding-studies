FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY server.go ./
RUN CGO_ENABLED=0 go build -o /out/server server.go

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/server /server
EXPOSE 9001
USER nonroot
ENTRYPOINT ["/server"]
