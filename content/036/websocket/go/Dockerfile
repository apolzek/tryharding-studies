FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download && go mod tidy
COPY server.go ./
RUN CGO_ENABLED=0 go build -o /out/server server.go

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/server /server
EXPOSE 8001
USER nonroot
ENTRYPOINT ["/server"]
